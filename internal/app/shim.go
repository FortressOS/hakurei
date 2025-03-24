package app

import (
	"context"
	"encoding/gob"
	"errors"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"git.gensokyo.uk/security/fortify/internal"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
	"git.gensokyo.uk/security/fortify/sandbox"
)

const shimEnv = "FORTIFY_SHIM"

type shimParams struct {
	// finalised container params
	Container *sandbox.Params
	// path to outer home directory
	Home string

	// verbosity pass through
	Verbose bool
}

// ShimMain is the main function of the shim process and runs as the unconstrained target user.
func ShimMain() {
	fmsg.Prepare("shim")

	if err := sandbox.SetDumpable(sandbox.SUID_DUMP_DISABLE); err != nil {
		log.Fatalf("cannot set SUID_DUMP_DISABLE: %s", err)
	}

	var (
		params     shimParams
		closeSetup func() error
	)
	if f, err := sandbox.Receive(shimEnv, &params, nil); err != nil {
		if errors.Is(err, sandbox.ErrInvalid) {
			log.Fatal("invalid config descriptor")
		}
		if errors.Is(err, sandbox.ErrNotSet) {
			log.Fatal("FORTIFY_SHIM not set")
		}

		log.Fatalf("cannot receive shim setup params: %v", err)
	} else {
		internal.InstallFmsg(params.Verbose)
		closeSetup = f
	}

	if params.Container == nil || params.Container.Ops == nil {
		log.Fatal("invalid container params")
	}

	// close setup socket
	if err := closeSetup(); err != nil {
		log.Printf("cannot close setup pipe: %v", err)
		// not fatal
	}

	// ensure home directory as target user
	if s, err := os.Stat(params.Home); err != nil {
		if os.IsNotExist(err) {
			if err = os.Mkdir(params.Home, 0700); err != nil {
				log.Fatalf("cannot create home directory: %v", err)
			}
		} else {
			log.Fatalf("cannot access home directory: %v", err)
		}

		// home directory is created, proceed
	} else if !s.IsDir() {
		log.Fatalf("path %q is not a directory", params.Home)
	}

	var name string
	if len(params.Container.Args) > 0 {
		name = params.Container.Args[0]
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop() // unreachable
	container := sandbox.New(ctx, name)
	container.Params = *params.Container
	container.Stdin, container.Stdout, container.Stderr = os.Stdin, os.Stdout, os.Stderr
	container.Cancel = func(cmd *exec.Cmd) error { return cmd.Process.Signal(os.Interrupt) }
	container.WaitDelay = 2 * time.Second

	if err := container.Start(); err != nil {
		fmsg.PrintBaseError(err, "cannot start container:")
		os.Exit(1)
	}
	if err := container.Serve(); err != nil {
		fmsg.PrintBaseError(err, "cannot configure container:")
	}
	if err := container.Wait(); err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) {
			if errors.Is(err, context.Canceled) {
				os.Exit(2)
			}
			log.Printf("wait: %v", err)
			os.Exit(127)
		}
		os.Exit(exitError.ExitCode())
	}
}

type shimProcess struct {
	// user switcher process
	cmd *exec.Cmd
	// fallback exit notifier with error returned killing the process
	killFallback chan error
	// monitor to shim encoder
	encoder *gob.Encoder
}

func (s *shimProcess) Unwrap() *exec.Cmd    { return s.cmd }
func (s *shimProcess) Fallback() chan error { return s.killFallback }

func (s *shimProcess) String() string {
	if s.cmd == nil {
		return "(unused shim manager)"
	}
	return s.cmd.String()
}

func (s *shimProcess) Start(
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
			shimEnv + "=" + strconv.Itoa(fd),
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

func (s *shimProcess) Serve(ctx context.Context, params *shimParams) error {
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
