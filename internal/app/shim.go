package app

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"

	"hakurei.app/container"
	"hakurei.app/container/seccomp"
	"hakurei.app/internal"
	"hakurei.app/internal/hlog"
)

//#include "shim-signal.h"
import "C"

const shimEnv = "HAKUREI_SHIM"

type shimParams struct {
	// monitor pid, checked against ppid in signal handler
	Monitor int

	// duration to wait for after interrupting a container's initial process before the container is killed;
	// zero value defaults to [DefaultShimWaitDelay], values exceeding [MaxShimWaitDelay] becomes [MaxShimWaitDelay]
	WaitDelay time.Duration

	// finalised container params
	Container *container.Params

	// verbosity pass through
	Verbose bool
}

const (
	// ShimExitRequest is returned when the monitor process requests shim exit.
	ShimExitRequest = 254
	// ShimExitOrphan is returned when the shim is orphaned before monitor delivers a signal.
	ShimExitOrphan = 3

	DefaultShimWaitDelay = 5 * time.Second
	MaxShimWaitDelay     = 30 * time.Second
)

// ShimMain is the main function of the shim process and runs as the unconstrained target user.
func ShimMain() {
	hlog.Prepare("shim")

	if err := container.SetDumpable(container.SUID_DUMP_DISABLE); err != nil {
		log.Fatalf("cannot set SUID_DUMP_DISABLE: %s", err)
	}

	var (
		params     shimParams
		closeSetup func() error
	)
	if f, err := container.Receive(shimEnv, &params, nil); err != nil {
		if errors.Is(err, syscall.EBADF) {
			log.Fatal("invalid config descriptor")
		}
		if errors.Is(err, container.ErrReceiveEnv) {
			log.Fatal("HAKUREI_SHIM not set")
		}

		log.Fatalf("cannot receive shim setup params: %v", err)
	} else {
		internal.InstallOutput(params.Verbose)
		closeSetup = f
	}

	var signalPipe io.ReadCloser
	// the Go runtime does not expose siginfo_t so SIGCONT is handled in C to check si_pid
	if r, w, err := os.Pipe(); err != nil {
		log.Fatalf("cannot pipe: %v", err)
	} else if _, err = C.hakurei_shim_setup_cont_signal(C.pid_t(params.Monitor), C.int(w.Fd())); err != nil {
		log.Fatalf("cannot install SIGCONT handler: %v", err)
	} else {
		defer runtime.KeepAlive(w)
		signalPipe = r
	}

	// pdeath_signal delivery is checked as if the dying process called kill(2), see kernel/exit.c
	if _, _, errno := syscall.Syscall(syscall.SYS_PRCTL, syscall.PR_SET_PDEATHSIG, uintptr(syscall.SIGCONT), 0); errno != 0 {
		log.Fatalf("cannot set parent-death signal: %v", errno)
	}

	// signal handler outcome
	var cancelContainer atomic.Pointer[context.CancelFunc]
	go func() {
		buf := make([]byte, 1)
		for {
			if _, err := signalPipe.Read(buf); err != nil {
				log.Fatalf("cannot read from signal pipe: %v", err)
			}

			switch buf[0] {
			case 0: // got SIGCONT from monitor: shim exit requested
				if fp := cancelContainer.Load(); params.Container.ForwardCancel && fp != nil && *fp != nil {
					(*fp)()
					// shim now bound by ShimWaitDelay, implemented below
					continue
				}

				// setup has not completed, terminate immediately
				hlog.Resume()
				os.Exit(ShimExitRequest)
				return

			case 1: // got SIGCONT after adoption: monitor died before delivering signal
				hlog.BeforeExit()
				os.Exit(ShimExitOrphan)
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

	if params.Container == nil || params.Container.Ops == nil {
		log.Fatal("invalid container params")
	}

	// close setup socket
	if err := closeSetup(); err != nil {
		log.Printf("cannot close setup pipe: %v", err)
		// not fatal
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	cancelContainer.Store(&stop)
	z := container.New(ctx)
	z.Params = *params.Container
	z.Stdin, z.Stdout, z.Stderr = os.Stdin, os.Stdout, os.Stderr

	z.WaitDelay = params.WaitDelay
	if z.WaitDelay == 0 {
		z.WaitDelay = DefaultShimWaitDelay
	}
	if z.WaitDelay > MaxShimWaitDelay {
		z.WaitDelay = MaxShimWaitDelay
	}

	if err := z.Start(); err != nil {
		hlog.PrintBaseError(err, "cannot start container:")
		os.Exit(1)
	}
	if err := z.Serve(); err != nil {
		hlog.PrintBaseError(err, "cannot configure container:")
	}

	if err := seccomp.Load(
		seccomp.Preset(seccomp.PresetStrict, seccomp.AllowMultiarch),
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
