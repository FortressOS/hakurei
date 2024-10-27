package shim

import (
	"errors"
	"net"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"git.ophivana.moe/security/fortify/acl"
	"git.ophivana.moe/security/fortify/internal/fmsg"
)

// used by the parent process

type Shim struct {
	// user switcher process
	cmd *exec.Cmd
	// uid of shim target user
	uid uint32
	// whether to check shim pid
	checkPid bool
	// user switcher executable path
	executable string
	// path to setup socket
	socket string
	// shim setup abort reason and completion
	abort     chan error
	abortErr  atomic.Pointer[error]
	abortOnce sync.Once
	// wayland mediation, nil if disabled
	wl *Wayland
	// shim setup payload
	payload *Payload
}

func New(executable string, uid uint32, socket string, wl *Wayland, payload *Payload, checkPid bool) *Shim {
	return &Shim{uid: uid, executable: executable, socket: socket, wl: wl, payload: payload, checkPid: checkPid}
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

type CommandBuilder func(shimEnv string) (args []string)

func (s *Shim) Start(f CommandBuilder) (*time.Time, error) {
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
	s.cmd = exec.Command(s.executable, f(EnvShim+"="+s.socket)...)
	s.cmd.Env = []string{}
	s.cmd.Stdin, s.cmd.Stdout, s.cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	s.cmd.Dir = "/"
	fmsg.VPrintln("starting shim via user switcher:", s.cmd)
	fmsg.Withhold() // withhold messages to stderr
	if err := s.cmd.Start(); err != nil {
		return nil, fmsg.WrapErrorSuffix(err,
			"cannot start user switcher:")
	}
	startTime := time.Now().UTC()

	// kill shim if something goes wrong and an error is returned
	killShim := func() {
		if err := s.cmd.Process.Signal(os.Interrupt); err != nil {
			fmsg.Println("cannot terminate shim on faulted setup:", err)
		}
	}
	defer func() { killShim() }()

	accept()
	conn := <-cf
	if conn == nil {
		return &startTime, fmsg.WrapErrorSuffix(*s.abortErr.Load(), "cannot accept call on setup socket:")
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
	} else if s.checkPid && cred.Pid != int32(s.cmd.Process.Pid) {
		fmsg.Printf("process %d tried to connect to shim setup socket, expecting shim %d",
			cred.Pid, s.cmd.Process.Pid)
		err = errors.New("compromised target user")
		s.Abort(err)
		return &startTime, err
	}

	// serve payload and wayland fd if enabled
	// this also closes the connection
	err := s.payload.serve(conn, s.wl)
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
					close(cf)
					return
				case <-accept:
					if conn, err0 := l.AcceptUnix(); err0 != nil {
						s.Abort(err0) // does not block, breaks loop
						cf <- nil     // receiver sees nil value and loads err0 stored during abort
					} else {
						cf <- conn
					}
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
