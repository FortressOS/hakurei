package shim

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strconv"
	"syscall"

	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/helper"
	"git.gensokyo.uk/security/fortify/helper/proc"
	"git.gensokyo.uk/security/fortify/helper/seccomp"
	"git.gensokyo.uk/security/fortify/internal"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
	init0 "git.gensokyo.uk/security/fortify/internal/priv/init"
)

// everything beyond this point runs as unconstrained target user
// proceed with caution!

func Main() {
	// sharing stdout with fortify
	// USE WITH CAUTION
	fmsg.SetPrefix("shim")

	// setting this prevents ptrace
	if err := internal.PR_SET_DUMPABLE__SUID_DUMP_DISABLE(); err != nil {
		fmsg.Fatalf("cannot set SUID_DUMP_DISABLE: %s", err)
		panic("unreachable")
	}

	// receive setup payload
	var (
		payload    Payload
		closeSetup func() error
	)
	if f, err := proc.Receive(Env, &payload); err != nil {
		if errors.Is(err, proc.ErrInvalid) {
			fmsg.Fatal("invalid config descriptor")
		}
		if errors.Is(err, proc.ErrNotSet) {
			fmsg.Fatal("FORTIFY_SHIM not set")
		}

		fmsg.Fatalf("cannot decode shim setup payload: %v", err)
		panic("unreachable")
	} else {
		fmsg.SetVerbose(payload.Verbose)
		closeSetup = f
	}

	if payload.Bwrap == nil {
		fmsg.Fatal("bwrap config not supplied")
	}

	// restore bwrap sync fd
	var syncFd *os.File
	if payload.Sync != nil {
		syncFd = os.NewFile(*payload.Sync, "sync")
	}

	// close setup socket
	if err := closeSetup(); err != nil {
		fmsg.Println("cannot close setup pipe:", err)
		// not fatal
	}

	// ensure home directory as target user
	if s, err := os.Stat(payload.Home); err != nil {
		if os.IsNotExist(err) {
			if err = os.Mkdir(payload.Home, 0700); err != nil {
				fmsg.Fatalf("cannot create home directory: %v", err)
			}
		} else {
			fmsg.Fatalf("cannot access home directory: %v", err)
		}

		// home directory is created, proceed
	} else if !s.IsDir() {
		fmsg.Fatalf("data path %q is not a directory", payload.Home)
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
			fmsg.Fatal("no command was specified and environment is unset")
		}
		if ic.Argv0, ok = payload.Bwrap.SetEnv["SHELL"]; !ok {
			fmsg.Fatal("no command was specified and $SHELL was unset")
		}

		ic.Argv = []string{ic.Argv0}
	}

	conf := payload.Bwrap

	var extraFiles []*os.File

	// serve setup payload
	if fd, encoder, err := proc.Setup(&extraFiles); err != nil {
		fmsg.Fatalf("cannot pipe: %v", err)
	} else {
		conf.SetEnv[init0.Env] = strconv.Itoa(fd)
		go func() {
			fmsg.VPrintln("transmitting config to init")
			if err = encoder.Encode(&ic); err != nil {
				fmsg.Fatalf("cannot transmit init config: %v", err)
			}
		}()
	}

	helper.BubblewrapName = payload.Exec[0] // resolved bwrap path by parent
	if fmsg.Verbose() {
		seccomp.CPrintln = fmsg.Println
	}
	if b, err := helper.NewBwrap(
		conf, path.Join(fst.Tmp, "sbin/init"),
		nil, func(int, int) []string { return make([]string, 0) },
		extraFiles,
		syncFd,
	); err != nil {
		fmsg.Fatalf("malformed sandbox config: %v", err)
	} else {
		b.Stdin(os.Stdin).Stdout(os.Stdout).Stderr(os.Stderr)
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop() // unreachable

		// run and pass through exit code
		if err = b.Start(ctx, false); err != nil {
			fmsg.Fatalf("cannot start target process: %v", err)
		} else if err = b.Wait(); err != nil {
			var exitError *exec.ExitError
			if !errors.As(err, &exitError) {
				fmsg.Println("wait:", err)
				fmsg.Exit(127)
				panic("unreachable")
			}
			fmsg.Exit(exitError.ExitCode())
			panic("unreachable")
		}
	}
}
