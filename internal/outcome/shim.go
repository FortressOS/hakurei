package outcome

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"

	"hakurei.app/container"
	"hakurei.app/container/comp"
	"hakurei.app/container/seccomp"
	"hakurei.app/hst"
	"hakurei.app/message"
)

//#include "shim-signal.h"
import "C"

const (
	/* hakurei requests shim exit */
	shimMsgExitRequested = C.HAKUREI_SHIM_EXIT_REQUESTED
	/* shim orphaned before hakurei delivers a signal */
	shimMsgOrphaned = C.HAKUREI_SHIM_ORPHAN
	/* unreachable */
	shimMsgInvalid = C.HAKUREI_SHIM_INVALID
	/* unexpected si_pid */
	shimMsgBadPID = C.HAKUREI_SHIM_BAD_PID
)

// setupContSignal sets up the SIGCONT signal handler for the cross-uid shim exit hack.
// The signal handler is implemented in C, signals can be processed by reading from the returned reader.
// The returned function must be called after all signal processing concludes.
func setupContSignal(pid int) (io.ReadCloser, func(), error) {
	if r, w, err := os.Pipe(); err != nil {
		return nil, nil, err
	} else if _, err = C.hakurei_shim_setup_cont_signal(C.pid_t(pid), C.int(w.Fd())); err != nil {
		_, _ = r.Close(), w.Close()
		return nil, nil, err
	} else {
		return r, func() { runtime.KeepAlive(w) }, nil
	}
}

// shimEnv is the name of the environment variable storing decimal representation of
// setup pipe fd for [container.Receive].
const shimEnv = "HAKUREI_SHIM"

// shimParams is embedded in outcomeState and transmitted from priv side to shim.
type shimParams struct {
	// Priv side pid, checked against ppid in signal handler for the syscall.SIGCONT hack.
	PrivPID int

	// Duration to wait for after the initial process receives os.Interrupt before the container is killed.
	// Limits are enforced on the priv side.
	WaitDelay time.Duration

	// Verbosity pass through from [message.Msg].
	Verbose bool

	// Outcome setup ops, contains setup state. Populated by outcome.finalise.
	Ops []outcomeOp
}

// valid checks shimParams to be safe for use.
func (p *shimParams) valid() bool { return p != nil && p.PrivPID > 0 }

// shimName is the prefix used by log.std in the shim process.
const shimName = "shim"

// Shim is called by the main function of the shim process and runs as the unconstrained target user.
// Shim does not return.
func Shim(msg message.Msg) {
	if msg == nil {
		msg = message.NewMsg(log.Default())
	}
	shimEntrypoint(direct{msg})
}

func shimEntrypoint(k syscallDispatcher) {
	msg := k.getMsg()
	if msg == nil {
		panic("attempting to call shimEntrypoint with nil msg")
	} else if logger := msg.GetLogger(); logger != nil {
		logger.SetPrefix(shimName + ": ")
		logger.SetFlags(0)
	}

	if err := k.setDumpable(container.SUID_DUMP_DISABLE); err != nil {
		k.fatalf("cannot set SUID_DUMP_DISABLE: %v", err)
	}

	var (
		state      outcomeState
		closeSetup func() error
	)
	if f, err := k.receive(shimEnv, &state, nil); err != nil {
		if errors.Is(err, syscall.EBADF) {
			k.fatal("invalid config descriptor")
		}
		if errors.Is(err, container.ErrReceiveEnv) {
			k.fatal(shimEnv + " not set")
		}

		k.fatalf("cannot receive shim setup params: %v", err)
	} else {
		msg.SwapVerbose(state.Shim.Verbose)
		closeSetup = f

		if err = state.populateLocal(k, msg); err != nil {
			printMessageError(func(v ...any) { k.fatal(fmt.Sprintln(v...)) },
				"cannot populate local state:", err)
		}
	}

	// the Go runtime does not expose siginfo_t so SIGCONT is handled in C to check si_pid
	var signalPipe io.ReadCloser
	if r, wKeepAlive, err := k.setupContSignal(state.Shim.PrivPID); err != nil {
		switch {
		case errors.As(err, new(*os.SyscallError)): // returned by os.Pipe
			k.fatal(err.Error())
			return

		case errors.As(err, new(syscall.Errno)): // returned by hakurei_shim_setup_cont_signal
			k.fatalf("cannot install SIGCONT handler: %v", err)
			return

		default: // unreachable
			k.fatalf("cannot set up exit request: %v", err)
			return
		}
	} else {
		defer wKeepAlive()
		signalPipe = r
	}

	// pdeath_signal delivery is checked as if the dying process called kill(2), see kernel/exit.c
	if err := k.prctl(syscall.PR_SET_PDEATHSIG, uintptr(syscall.SIGCONT), 0); err != nil {
		k.fatalf("cannot set parent-death signal: %v", err)
	}

	stateParams := state.newParams()
	for _, op := range state.Shim.Ops {
		if err := op.toContainer(stateParams); err != nil {
			printMessageError(func(v ...any) { k.fatal(fmt.Sprintln(v...)) },
				"cannot create container state:", err)
		}
	}
	if stateParams.params.Ops == nil { // only reachable with corrupted outcomeState
		k.fatal("invalid container state")
	}

	// shim exit outcomes
	var cancelContainer atomic.Pointer[context.CancelFunc]
	k.new(func(k syscallDispatcher, msg message.Msg) {
		buf := make([]byte, 1)
		for {
			if _, err := signalPipe.Read(buf); err != nil {
				k.fatalf("cannot read from signal pipe: %v", err)
			}

			switch buf[0] {
			case shimMsgExitRequested: // got SIGCONT from hakurei: shim exit requested
				if fp := cancelContainer.Load(); stateParams.params.ForwardCancel && fp != nil && *fp != nil {
					(*fp)()
					// shim now bound by ShimWaitDelay, implemented below
					continue
				}

				// setup has not completed, terminate immediately
				k.exit(hst.ExitRequest)

			case shimMsgOrphaned: // got SIGCONT after orphaned: hakurei died before delivering signal
				k.exit(hst.ExitOrphan)

			case shimMsgInvalid: // unreachable
				msg.Verbose("sa_sigaction got invalid siginfo")

			case shimMsgBadPID: // got SIGCONT from unexpected process: hopefully the terminal driver
				msg.Verbose("got SIGCONT from unexpected process")

			default: // unreachable
				k.fatalf("got invalid message %d from signal handler", buf[0])
			}
		}
	})

	// close setup socket
	if err := closeSetup(); err != nil {
		msg.Verbosef("cannot close setup pipe: %v", err)
		// not fatal
	}

	ctx, stop := k.notifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	cancelContainer.Store(&stop)
	z := container.New(ctx, msg)
	z.Params = *stateParams.params
	z.Stdin, z.Stdout, z.Stderr = os.Stdin, os.Stdout, os.Stderr

	// bounds and default enforced in finalise.go
	z.WaitDelay = state.Shim.WaitDelay

	if err := k.containerStart(z); err != nil {
		var f func(v ...any)
		if logger := msg.GetLogger(); logger != nil {
			f = logger.Println
		} else {
			f = func(v ...any) {
				msg.Verbose(fmt.Sprintln(v...))
			}
		}
		printMessageError(f, "cannot start container:", err)
		k.exit(hst.ExitFailure)
	}
	if err := k.containerServe(z); err != nil {
		printMessageError(func(v ...any) { k.fatal(fmt.Sprintln(v...)) },
			"cannot configure container:", err)
	}

	if err := k.seccompLoad(
		seccomp.Preset(comp.PresetStrict, seccomp.AllowMultiarch),
		seccomp.AllowMultiarch,
	); err != nil {
		k.fatalf("cannot load syscall filter: %v", err)
	}

	if err := k.containerWait(z); err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) {
			if errors.Is(err, context.Canceled) {
				k.exit(hst.ExitCancel)
			}
			msg.Verbosef("cannot wait: %v", err)
			k.exit(127)
		}
		k.exit(exitError.ExitCode())
	}
}
