package shim

import (
	"encoding/gob"
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	shim0 "git.gensokyo.uk/security/fortify/cmd/fshim/ipc"
	"git.gensokyo.uk/security/fortify/internal"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
	"git.gensokyo.uk/security/fortify/internal/proc"
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
	// fallback exit notifier with error returned killing the process
	killFallback chan error
	// shim setup payload
	payload *shim0.Payload
}

func New(uid uint32, aid string, supp []string, payload *shim0.Payload) *Shim {
	return &Shim{uid: uid, aid: aid, supp: supp, payload: payload}
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

func (s *Shim) WaitFallback() chan error {
	return s.killFallback
}

func (s *Shim) Start() (*time.Time, error) {
	// start user switcher process and save time
	var fsu string
	if p, ok := internal.Check(internal.Fsu); !ok {
		fmsg.Fatal("invalid fsu path, this copy of fshim is not compiled correctly")
		panic("unreachable")
	} else {
		fsu = p
	}
	s.cmd = exec.Command(fsu)

	var encoder *gob.Encoder
	if fd, e, err := proc.Setup(&s.cmd.ExtraFiles); err != nil {
		return nil, fmsg.WrapErrorSuffix(err,
			"cannot create shim setup pipe:")
	} else {
		encoder = e
		s.cmd.Env = []string{
			shim0.Env + "=" + strconv.Itoa(fd),
			"FORTIFY_APP_ID=" + s.aid,
		}
	}

	if len(s.supp) > 0 {
		fmsg.VPrintf("attaching supplementary group ids %s", s.supp)
		s.cmd.Env = append(s.cmd.Env, "FORTIFY_GROUPS="+strings.Join(s.supp, " "))
	}
	s.cmd.Stdin, s.cmd.Stdout, s.cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	s.cmd.Dir = "/"

	// pass sync fd if set
	if s.payload.Bwrap.Sync() != nil {
		fd := proc.ExtraFile(s.cmd, s.payload.Bwrap.Sync())
		s.payload.Sync = &fd
	}

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

	// take alternative exit path on signal
	sig := make(chan os.Signal, 2)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		v := <-sig
		fmsg.Printf("got %s after program start", v)
		s.killFallback <- nil
		signal.Ignore(syscall.SIGINT, syscall.SIGTERM)
	}()

	shimErr := make(chan error)
	go func() { shimErr <- encoder.Encode(s.payload) }()

	select {
	case err := <-shimErr:
		if err != nil {
			return &startTime, fmsg.WrapErrorSuffix(err,
				"cannot transmit shim config:")
		}
		killShim = func() {}
	case <-time.After(shimSetupTimeout):
		return &startTime, fmsg.WrapError(errors.New("timed out waiting for shim"),
			"timed out waiting for shim")
	}

	return &startTime, nil
}
