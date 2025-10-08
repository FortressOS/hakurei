package app

import (
	"context"
	"errors"
	"io"
	"log"
	"maps"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"

	"hakurei.app/container"
	"hakurei.app/container/bits"
	"hakurei.app/container/seccomp"
	"hakurei.app/hst"
)

//#include "shim-signal.h"
import "C"

const (
	// setup pipe fd for [container.Receive]
	shimEnv = "HAKUREI_SHIM"

	// only used for a nil configured env map
	envAllocSize = 1 << 6
)

type shimParams struct {
	// Priv side pid, checked against ppid in signal handler for the syscall.SIGCONT hack.
	PrivPID int

	// Duration to wait for after the initial process receives os.Interrupt before the container is killed.
	// Limits are enforced on the priv side.
	WaitDelay time.Duration

	// Verbosity pass through from [container.Msg].
	Verbose bool

	// Outcome setup ops, contains setup state. Populated by outcome.finalise.
	Ops []outcomeOp
}

// valid checks shimParams to be safe for use.
func (p *shimParams) valid() bool {
	return p != nil &&
		p.Ops != nil &&
		p.PrivPID > 0
}

// ShimMain is the main function of the shim process and runs as the unconstrained target user.
func ShimMain() {
	log.SetPrefix("shim: ")
	log.SetFlags(0)
	msg := container.NewMsg(log.Default())

	if err := container.SetDumpable(container.SUID_DUMP_DISABLE); err != nil {
		log.Fatalf("cannot set SUID_DUMP_DISABLE: %s", err)
	}

	var (
		state      outcomeState
		closeSetup func() error
	)
	if f, err := container.Receive(shimEnv, &state, nil); err != nil {
		if errors.Is(err, syscall.EBADF) {
			log.Fatal("invalid config descriptor")
		}
		if errors.Is(err, container.ErrReceiveEnv) {
			log.Fatal(shimEnv + " not set")
		}

		log.Fatalf("cannot receive shim setup params: %v", err)
	} else {
		msg.SwapVerbose(state.Shim.Verbose)
		closeSetup = f

		if err = state.populateLocal(direct{}, msg); err != nil {
			if m, ok := container.GetErrorMessage(err); ok {
				log.Fatal(m)
			} else {
				log.Fatalf("cannot populate local state: %v", err)
			}
		}
	}

	// the Go runtime does not expose siginfo_t so SIGCONT is handled in C to check si_pid
	var signalPipe io.ReadCloser
	if r, w, err := os.Pipe(); err != nil {
		log.Fatalf("cannot pipe: %v", err)
	} else if _, err = C.hakurei_shim_setup_cont_signal(C.pid_t(state.Shim.PrivPID), C.int(w.Fd())); err != nil {
		log.Fatalf("cannot install SIGCONT handler: %v", err)
	} else {
		defer runtime.KeepAlive(w)
		signalPipe = r
	}

	// pdeath_signal delivery is checked as if the dying process called kill(2), see kernel/exit.c
	if _, _, errno := syscall.Syscall(syscall.SYS_PRCTL, syscall.PR_SET_PDEATHSIG, uintptr(syscall.SIGCONT), 0); errno != 0 {
		log.Fatalf("cannot set parent-death signal: %v", errno)
	}

	var params container.Params
	stateParams := outcomeStateParams{params: &params, outcomeState: &state}
	if state.Container.Env == nil {
		stateParams.env = make(map[string]string, envAllocSize)
	} else {
		stateParams.env = maps.Clone(state.Container.Env)
	}
	for _, op := range state.Shim.Ops {
		if err := op.toContainer(&stateParams); err != nil {
			if m, ok := container.GetErrorMessage(err); ok {
				log.Fatal(m)
			} else {
				log.Fatalf("cannot create container state: %v", err)
			}
		}
	}

	// shim exit outcomes
	var cancelContainer atomic.Pointer[context.CancelFunc]
	go func() {
		buf := make([]byte, 1)
		for {
			if _, err := signalPipe.Read(buf); err != nil {
				log.Fatalf("cannot read from signal pipe: %v", err)
			}

			switch buf[0] {
			case 0: // got SIGCONT from monitor: shim exit requested
				if fp := cancelContainer.Load(); params.ForwardCancel && fp != nil && *fp != nil {
					(*fp)()
					// shim now bound by ShimWaitDelay, implemented below
					continue
				}

				// setup has not completed, terminate immediately
				msg.Resume()
				os.Exit(hst.ShimExitRequest)
				return

			case 1: // got SIGCONT after adoption: monitor died before delivering signal
				msg.BeforeExit()
				os.Exit(hst.ShimExitOrphan)
				return

			case 2: // unreachable
				log.Println("sa_sigaction got invalid siginfo")

			case 3: // got SIGCONT from unexpected process: hopefully the terminal driver
				log.Println("got SIGCONT from unexpected process")

			default: // unreachable
				log.Fatalf("got invalid message %d from signal handler", buf[0])
			}
		}
	}()

	if params.Ops == nil {
		log.Fatal("invalid container params")
	}

	// close setup socket
	if err := closeSetup(); err != nil {
		log.Printf("cannot close setup pipe: %v", err)
		// not fatal
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	cancelContainer.Store(&stop)
	z := container.New(ctx, msg)
	z.Params = params
	z.Stdin, z.Stdout, z.Stderr = os.Stdin, os.Stdout, os.Stderr

	// bounds and default enforced in finalise.go
	z.WaitDelay = state.Shim.WaitDelay

	if err := z.Start(); err != nil {
		printMessageError("cannot start container:", err)
		os.Exit(1)
	}
	if err := z.Serve(); err != nil {
		printMessageError("cannot configure container:", err)
	}

	if err := seccomp.Load(
		seccomp.Preset(bits.PresetStrict, seccomp.AllowMultiarch),
		seccomp.AllowMultiarch,
	); err != nil {
		log.Fatalf("cannot load syscall filter: %v", err)
	}

	if err := z.Wait(); err != nil {
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
