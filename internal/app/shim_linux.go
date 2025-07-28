package app

import (
	"context"
	"errors"
	"log"
	"os"
	"os/exec"
	"os/signal"
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

	// finalised container params
	Container *container.Params
	// path to outer home directory
	Home string

	// verbosity pass through
	Verbose bool
}

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
		if errors.Is(err, container.ErrInvalid) {
			log.Fatal("invalid config descriptor")
		}
		if errors.Is(err, container.ErrNotSet) {
			log.Fatal("HAKUREI_SHIM not set")
		}

		log.Fatalf("cannot receive shim setup params: %v", err)
	} else {
		internal.InstallOutput(params.Verbose)
		closeSetup = f

		// the Go runtime does not expose siginfo_t so SIGCONT is handled in C to check si_pid
		if _, err = C.hakurei_shim_setup_cont_signal(C.pid_t(params.Monitor)); err != nil {
			log.Fatalf("cannot install SIGCONT handler: %v", err)
		}

		// pdeath_signal delivery is checked as if the dying process called kill(2), see kernel/exit.c
		if _, _, errno := syscall.Syscall(syscall.SYS_PRCTL, syscall.PR_SET_PDEATHSIG, uintptr(syscall.SIGCONT), 0); errno != 0 {
			log.Fatalf("cannot set parent-death signal: %v", errno)
		}
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
	z := container.New(ctx, name)
	z.Params = *params.Container
	z.Stdin, z.Stdout, z.Stderr = os.Stdin, os.Stdout, os.Stderr
	z.Cancel = func(cmd *exec.Cmd) error { return cmd.Process.Signal(os.Interrupt) }
	z.WaitDelay = 2 * time.Second

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
