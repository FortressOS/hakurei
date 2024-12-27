package main

import (
	"errors"
	"os"
	"path"
	"strconv"
	"syscall"

	init0 "git.gensokyo.uk/security/fortify/cmd/finit/ipc"
	shim "git.gensokyo.uk/security/fortify/cmd/fshim/ipc"
	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/helper"
	"git.gensokyo.uk/security/fortify/internal"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
	"git.gensokyo.uk/security/fortify/internal/proc"
)

// everything beyond this point runs as unconstrained target user
// proceed with caution!

func main() {
	// sharing stdout with fortify
	// USE WITH CAUTION
	fmsg.SetPrefix("shim")

	// setting this prevents ptrace
	if err := internal.PR_SET_DUMPABLE__SUID_DUMP_DISABLE(); err != nil {
		fmsg.Fatalf("cannot set SUID_DUMP_DISABLE: %s", err)
		panic("unreachable")
	}

	// re-exec
	if len(os.Args) > 0 && (os.Args[0] != "fshim" || len(os.Args) != 1) && path.IsAbs(os.Args[0]) {
		if err := syscall.Exec(os.Args[0], []string{"fshim"}, os.Environ()); err != nil {
			fmsg.Println("cannot re-exec self:", err)
			// continue anyway
		}
	}

	// check path to finit
	var finitPath string
	if p, ok := internal.Path(internal.Finit); !ok {
		fmsg.Fatal("invalid finit path, this copy of fshim is not compiled correctly")
	} else {
		finitPath = p
	}

	// receive setup payload
	var (
		payload    shim.Payload
		closeSetup func() error
	)
	if f, err := proc.Receive(shim.Env, &payload); err != nil {
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
	if payload.Sync != nil {
		payload.Bwrap.SetSync(os.NewFile(*payload.Sync, "sync"))
	}

	// close setup socket
	if err := closeSetup(); err != nil {
		fmsg.Println("cannot close setup pipe:", err)
		// not fatal
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

	// bind finit inside sandbox
	finitInnerPath := path.Join(fst.Tmp, "sbin", "init")
	conf.Bind(finitPath, finitInnerPath)

	helper.BubblewrapName = payload.Exec[0] // resolved bwrap path by parent
	if b, err := helper.NewBwrap(conf, nil, finitInnerPath,
		func(int, int) []string { return make([]string, 0) }); err != nil {
		fmsg.Fatalf("malformed sandbox config: %v", err)
	} else {
		cmd := b.Unwrap()
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		cmd.ExtraFiles = extraFiles

		if fmsg.Verbose() {
			fmsg.VPrintln("bwrap args:", conf.Args())
		}

		// run and pass through exit code
		if err = b.Start(); err != nil {
			fmsg.Fatalf("cannot start target process: %v", err)
		} else if err = b.Wait(); err != nil {
			fmsg.VPrintln("wait:", err)
		}
		if b.Unwrap().ProcessState != nil {
			fmsg.Exit(b.Unwrap().ProcessState.ExitCode())
		} else {
			fmsg.Exit(127)
		}
	}
}
