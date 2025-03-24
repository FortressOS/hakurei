package shim

import (
	"context"
	"encoding/gob"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"git.gensokyo.uk/security/fortify/internal"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
	"git.gensokyo.uk/security/fortify/sandbox"
)

// used by the parent process

type Shim struct {
	// user switcher process
	cmd *exec.Cmd
	// fallback exit notifier with error returned killing the process
	killFallback chan error
	// monitor to shim encoder
	encoder *gob.Encoder
}

func (s *Shim) Unwrap() *exec.Cmd    { return s.cmd }
func (s *Shim) Fallback() chan error { return s.killFallback }

func (s *Shim) String() string {
	if s.cmd == nil {
		return "(unused shim manager)"
	}
	return s.cmd.String()
}

func (s *Shim) Start(
	aid string,
	supp []string,
) (*time.Time, error) {
	// prepare user switcher invocation
	fsuPath := internal.MustFsuPath()
	s.cmd = exec.Command(fsuPath)

	// pass shim setup pipe
	if fd, e, err := sandbox.Setup(&s.cmd.ExtraFiles); err != nil {
		return nil, fmsg.WrapErrorSuffix(err,
			"cannot create shim setup pipe:")
	} else {
		s.encoder = e
		s.cmd.Env = []string{
			Env + "=" + strconv.Itoa(fd),
			"FORTIFY_APP_ID=" + aid,
		}
	}

	// format fsu supplementary groups
	if len(supp) > 0 {
		fmsg.Verbosef("attaching supplementary group ids %s", supp)
		s.cmd.Env = append(s.cmd.Env, "FORTIFY_GROUPS="+strings.Join(supp, " "))
	}
	s.cmd.Stdin, s.cmd.Stdout, s.cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	s.cmd.Dir = "/"

	fmsg.Verbose("starting shim via fsu:", s.cmd)
	// withhold messages to stderr
	fmsg.Suspend()
	if err := s.cmd.Start(); err != nil {
		return nil, fmsg.WrapErrorSuffix(err,
			"cannot start fsu:")
	}
	startTime := time.Now().UTC()

	return &startTime, nil
}

func (s *Shim) Serve(ctx context.Context, params *Params) error {
	// kill shim if something goes wrong and an error is returned
	s.killFallback = make(chan error, 1)
	killShim := func() {
		if err := s.cmd.Process.Signal(os.Interrupt); err != nil {
			s.killFallback <- err
		}
	}
	defer func() { killShim() }()

	encodeErr := make(chan error)
	go func() { encodeErr <- s.encoder.Encode(params) }()

	select {
	// encode return indicates setup completion
	case err := <-encodeErr:
		if err != nil {
			return fmsg.WrapErrorSuffix(err,
				"cannot transmit shim config:")
		}
		killShim = func() {}
		return nil

	// setup canceled before payload was accepted
	case <-ctx.Done():
		err := ctx.Err()
		if errors.Is(err, context.Canceled) {
			return fmsg.WrapError(syscall.ECANCELED,
				"shim setup canceled")
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return fmsg.WrapError(syscall.ETIMEDOUT,
				"deadline exceeded waiting for shim")
		}
		// unreachable
		return err
	}
}
