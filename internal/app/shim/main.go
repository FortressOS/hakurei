package shim

import (
	"context"
	"errors"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"git.gensokyo.uk/security/fortify/internal"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
	"git.gensokyo.uk/security/fortify/sandbox"
)

const Env = "FORTIFY_SHIM"

type Params struct {
	// finalised container params
	Container *sandbox.Params
	// path to outer home directory
	Home string

	// verbosity pass through
	Verbose bool
}

// everything beyond this point runs as unconstrained target user
// proceed with caution!

func Main() {
	// sharing stdout with fortify
	// USE WITH CAUTION
	fmsg.Prepare("shim")

	if err := sandbox.SetDumpable(sandbox.SUID_DUMP_DISABLE); err != nil {
		log.Fatalf("cannot set SUID_DUMP_DISABLE: %s", err)
	}

	var (
		params     Params
		closeSetup func() error
	)
	if f, err := sandbox.Receive(Env, &params, nil); err != nil {
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
