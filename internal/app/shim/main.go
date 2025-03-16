package shim

import (
	"context"
	"errors"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strconv"
	"syscall"

	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/helper"
	"git.gensokyo.uk/security/fortify/internal"
	"git.gensokyo.uk/security/fortify/internal/app/init0"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
	"git.gensokyo.uk/security/fortify/sandbox"
)

// everything beyond this point runs as unconstrained target user
// proceed with caution!

func Main() {
	// sharing stdout with fortify
	// USE WITH CAUTION
	fmsg.Prepare("shim")

	// setting this prevents ptrace
	if err := sandbox.SetDumpable(sandbox.SUID_DUMP_DISABLE); err != nil {
		log.Fatalf("cannot set SUID_DUMP_DISABLE: %s", err)
	}

	// receive setup payload
	var (
		payload    Payload
		closeSetup func() error
	)
	if f, err := sandbox.Receive(Env, &payload, nil); err != nil {
		if errors.Is(err, sandbox.ErrInvalid) {
			log.Fatal("invalid config descriptor")
		}
		if errors.Is(err, sandbox.ErrNotSet) {
			log.Fatal("FORTIFY_SHIM not set")
		}

		log.Fatalf("cannot decode shim setup payload: %v", err)
	} else {
		internal.InstallFmsg(payload.Verbose)
		closeSetup = f
	}

	if payload.Bwrap == nil {
		log.Fatal("bwrap config not supplied")
	}

	// restore bwrap sync fd
	var syncFd *os.File
	if payload.Sync != nil {
		syncFd = os.NewFile(*payload.Sync, "sync")
	}

	// close setup socket
	if err := closeSetup(); err != nil {
		log.Println("cannot close setup pipe:", err)
		// not fatal
	}

	// ensure home directory as target user
	if s, err := os.Stat(payload.Home); err != nil {
		if os.IsNotExist(err) {
			if err = os.Mkdir(payload.Home, 0700); err != nil {
				log.Fatalf("cannot create home directory: %v", err)
			}
		} else {
			log.Fatalf("cannot access home directory: %v", err)
		}

		// home directory is created, proceed
	} else if !s.IsDir() {
		log.Fatalf("data path %q is not a directory", payload.Home)
	}

	var ic init0.Payload

	// resolve argv0
	ic.Argv = payload.Argv
	if len(ic.Argv) > 0 {
		// looked up from $PATH by parent
		ic.Argv0 = payload.Exec[1]
	} else {
		// no argv, look up shell instead
		var ok bool
		if payload.Bwrap.SetEnv == nil {
			log.Fatal("no command was specified and environment is unset")
		}
		if ic.Argv0, ok = payload.Bwrap.SetEnv["SHELL"]; !ok {
			log.Fatal("no command was specified and $SHELL was unset")
		}

		ic.Argv = []string{ic.Argv0}
	}

	conf := payload.Bwrap

	var extraFiles []*os.File

	// serve setup payload
	if fd, encoder, err := sandbox.Setup(&extraFiles); err != nil {
		log.Fatalf("cannot pipe: %v", err)
	} else {
		conf.SetEnv[init0.Env] = strconv.Itoa(fd)
		go func() {
			fmsg.Verbose("transmitting config to init")
			if err = encoder.Encode(&ic); err != nil {
				log.Fatalf("cannot transmit init config: %v", err)
			}
		}()
	}

	helper.BubblewrapName = payload.Exec[0] // resolved bwrap path by parent

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop() // unreachable
	if b, err := helper.NewBwrap(
		ctx, path.Join(fst.Tmp, "sbin/init0"),
		nil, false,
		func(int, int) []string { return make([]string, 0) },
		func(cmd *exec.Cmd) { cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr },
		extraFiles,
		conf, syncFd,
	); err != nil {
		log.Fatalf("malformed sandbox config: %v", err)
	} else {
		// run and pass through exit code
		if err = b.Start(); err != nil {
			log.Fatalf("cannot start target process: %v", err)
		} else if err = b.Wait(); err != nil {
			var exitError *exec.ExitError
			if !errors.As(err, &exitError) {
				log.Printf("wait: %v", err)
				internal.Exit(127)
				panic("unreachable")
			}
			internal.Exit(exitError.ExitCode())
			panic("unreachable")
		}
	}
}
