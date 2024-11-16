package shim

import (
	"errors"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"git.ophivana.moe/security/fortify/acl"
	shim0 "git.ophivana.moe/security/fortify/cmd/fshim/ipc"
	"git.ophivana.moe/security/fortify/internal"
	"git.ophivana.moe/security/fortify/internal/fmsg"
)

const shimSetupTimeout = 5 * time.Second

// used by the parent process

type Shim struct {
	// user switcher process
	cmd *exec.Cmd
	// uid of shim target user
	uid uint32
	// string representation of application id
	aid string
	// string representation of supplementary group ids
	supp []string
	// path to setup socket
	socket string
	// shim setup abort reason and completion
	abort     chan error
	abortErr  atomic.Pointer[error]
	abortOnce sync.Once
	// fallback exit notifier with error returned killing the process
	killFallback chan error
	// wayland mediation, nil if disabled
	wl *shim0.Wayland
	// shim setup payload
	payload *shim0.Payload
}

func New(uid uint32, aid string, supp []string, socket string, wl *shim0.Wayland, payload *shim0.Payload) *Shim {
	return &Shim{uid: uid, aid: aid, supp: supp, socket: socket, wl: wl, payload: payload}
}

func (s *Shim) String() string {
	if s.cmd == nil {
		return "(unused shim manager)"
	}
	return s.cmd.String()
}

func (s *Shim) Unwrap() *exec.Cmd {
	return s.cmd
}

func (s *Shim) Abort(err error) {
	s.abortOnce.Do(func() {
		s.abortErr.Store(&err)
		// s.abort is buffered so this will never block
		s.abort <- err
	})
}

func (s *Shim) AbortWait(err error) {
	s.Abort(err)
	<-s.abort
}

func (s *Shim) WaitFallback() chan error {
	return s.killFallback
}

func (s *Shim) Start() (*time.Time, error) {
	var (
		cf     chan *net.UnixConn
		accept func()
	)

	// listen on setup socket
	if c, a, err := s.serve(); err != nil {
		return nil, fmsg.WrapErrorSuffix(err,
			"cannot listen on shim setup socket:")
	} else {
		// accepts a connection after each call to accept
		// connections are sent to the channel cf
		cf, accept = c, a
	}

	// start user switcher process and save time
	var fsu string
	if p, ok := internal.Check(internal.Fsu); !ok {
		fmsg.Fatal("invalid fsu path, this copy of fshim is not compiled correctly")
		panic("unreachable")
	} else {
		fsu = p
	}
	s.cmd = exec.Command(fsu)
	s.cmd.Env = []string{
		shim0.Env + "=" + s.socket,
		"FORTIFY_APP_ID=" + s.aid,
	}
	if len(s.supp) > 0 {
		fmsg.VPrintf("attaching supplementary group ids %s", s.supp)
		s.cmd.Env = append(s.cmd.Env, "FORTIFY_GROUPS="+strings.Join(s.supp, " "))
	}
	s.cmd.Stdin, s.cmd.Stdout, s.cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	s.cmd.Dir = "/"
	fmsg.VPrintln("starting shim via fsu:", s.cmd)
	fmsg.Suspend() // withhold messages to stderr
	if err := s.cmd.Start(); err != nil {
		return nil, fmsg.WrapErrorSuffix(err,
			"cannot start fsu:")
	}
	startTime := time.Now().UTC()

	// kill shim if something goes wrong and an error is returned
	s.killFallback = make(chan error, 1)
	killShim := func() {
		if err := s.cmd.Process.Signal(os.Interrupt); err != nil {
			s.killFallback <- err
		}
	}
	defer func() { killShim() }()

	accept()
	var conn *net.UnixConn
	select {
	case c := <-cf:
		if c == nil {
			return &startTime, fmsg.WrapErrorSuffix(*s.abortErr.Load(), "cannot accept call on setup socket:")
		} else {
			conn = c
		}
	case <-time.After(shimSetupTimeout):
		err := fmsg.WrapError(errors.New("timed out waiting for shim"),
			"timed out waiting for shim to connect")
		s.AbortWait(err)
		return &startTime, err
	}

	// authenticate against called provided uid and shim pid
	if cred, err := peerCred(conn); err != nil {
		return &startTime, fmsg.WrapErrorSuffix(*s.abortErr.Load(), "cannot retrieve shim credentials:")
	} else if cred.Uid != s.uid {
		fmsg.Printf("process %d owned by user %d tried to connect, expecting %d",
			cred.Pid, cred.Uid, s.uid)
		err = errors.New("compromised fortify build")
		s.Abort(err)
		return &startTime, err
	} else if cred.Pid != int32(s.cmd.Process.Pid) {
		fmsg.Printf("process %d tried to connect to shim setup socket, expecting shim %d",
			cred.Pid, s.cmd.Process.Pid)
		err = errors.New("compromised target user")
		s.Abort(err)
		return &startTime, err
	}

	// serve payload and wayland fd if enabled
	// this also closes the connection
	err := s.payload.Serve(conn, s.wl)
	if err == nil {
		killShim = func() {}
	}
	s.Abort(err) // aborting with nil indicates success
	return &startTime, err
}

func (s *Shim) serve() (chan *net.UnixConn, func(), error) {
	if s.abort != nil {
		panic("attempted to serve shim setup twice")
	}
	s.abort = make(chan error, 1)

	cf := make(chan *net.UnixConn)
	accept := make(chan struct{}, 1)

	if l, err := net.ListenUnix("unix", &net.UnixAddr{Name: s.socket, Net: "unix"}); err != nil {
		return nil, nil, err
	} else {
		l.SetUnlinkOnClose(true)

		fmsg.VPrintf("listening on shim setup socket %q", s.socket)
		if err = acl.UpdatePerm(s.socket, int(s.uid), acl.Read, acl.Write, acl.Execute); err != nil {
			fmsg.Println("cannot append ACL entry to shim setup socket:", err)
			s.Abort(err) // ensures setup socket cleanup
		}

		go func() {
			cfWg := new(sync.WaitGroup)
			for {
				select {
				case err = <-s.abort:
					if err != nil {
						fmsg.VPrintln("aborting shim setup, reason:", err)
					}
					if err = l.Close(); err != nil {
						fmsg.Println("cannot close setup socket:", err)
					}
					close(s.abort)
					go func() {
						cfWg.Wait()
						close(cf)
					}()
					return
				case <-accept:
					cfWg.Add(1)
					go func() {
						defer cfWg.Done()
						if conn, err0 := l.AcceptUnix(); err0 != nil {
							// breaks loop
							s.Abort(err0)
							// receiver sees nil value and loads err0 stored during abort
							cf <- nil
						} else {
							cf <- conn
						}
					}()
				}
			}
		}()
	}

	return cf, func() { accept <- struct{}{} }, nil
}

// peerCred fetches peer credentials of conn
func peerCred(conn *net.UnixConn) (ucred *syscall.Ucred, err error) {
	var raw syscall.RawConn
	if raw, err = conn.SyscallConn(); err != nil {
		return
	}

	err0 := raw.Control(func(fd uintptr) {
		ucred, err = syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	})
	err = errors.Join(err, err0)
	return
}
