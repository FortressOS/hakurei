package shim

import (
	"errors"
	"flag"
	"io"
	"os"
	"path"
	"strconv"

	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/helper"
	"git.gensokyo.uk/security/fortify/helper/bwrap"
	"git.gensokyo.uk/security/fortify/internal"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
	"git.gensokyo.uk/security/fortify/internal/proc"
	init0 "git.gensokyo.uk/security/fortify/internal/proc/priv/init"
)

// everything beyond this point runs as unconstrained target user
// proceed with caution!

func Main(args []string) {
	// sharing stdout with fortify
	// USE WITH CAUTION
	fmsg.SetPrefix("shim")

	// setting this prevents ptrace
	if err := internal.PR_SET_DUMPABLE__SUID_DUMP_DISABLE(); err != nil {
		fmsg.Fatalf("cannot set SUID_DUMP_DISABLE: %s", err)
		panic("unreachable")
	}

	set := flag.NewFlagSet("shim", flag.ExitOnError)

	// debug: export seccomp filter
	debugExportSeccomp := set.String("export-seccomp", "", "export the seccomp filter to file")
	debugExportSeccompFlags := [...]struct {
		o syscallOpts
		v *bool
	}{
		{flagDenyNS, set.Bool("deny-ns", false, "deny namespace-related syscalls")},
		{flagDenyTTY, set.Bool("deny-tty", false, "deny faking input ioctls")},
		{flagDenyDevel, set.Bool("deny-devel", false, "deny development syscalls")},
		{flagMultiarch, set.Bool("multiarch", false, "allow multiarch")},
		{flagLinux32, set.Bool("linux32", false, "allow PER_LINUX32")},
		{flagCan, set.Bool("can", false, "allow AF_CAN")},
		{flagBluetooth, set.Bool("bluetooth", false, "AF_BLUETOOTH")},
	}

	// Ignore errors; set is set for ExitOnError.
	_ = set.Parse(args[1:])

	// debug: export seccomp filter
	if *debugExportSeccomp != "" {
		var opts syscallOpts
		for _, opt := range debugExportSeccompFlags {
			if *opt.v {
				opts |= opt.o
			}
		}

		if f, err := os.Create(*debugExportSeccomp); err != nil {
			fmsg.Fatalf("cannot create %q: %v", *debugExportSeccomp, err)
		} else {
			mustExportFilter(f, opts)
			if err = f.Close(); err != nil {
				fmsg.Fatalf("cannot close %q: %v", *debugExportSeccomp, err)
			}
		}
		fmsg.Exit(0)
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

	// bind fortify inside sandbox
	var (
		innerSbin    = path.Join(fst.Tmp, "sbin")
		innerFortify = path.Join(innerSbin, "fortify")
		innerInit    = path.Join(innerSbin, "init")
	)
	conf.Bind(proc.MustExecutable(), innerFortify)
	conf.Symlink("fortify", innerInit)

	helper.BubblewrapName = payload.Exec[0] // resolved bwrap path by parent
	if b, err := helper.NewBwrap(
		conf, innerInit,
		nil, func(int, int) []string { return make([]string, 0) },
		[]helper.BwrapExtraFile{
			// keep this fd open while sandbox is running
			// (--sync-fd FD)
			{"--sync-fd", syncFd},
			// load and use seccomp rules from FD (not repeatable)
			// (--seccomp FD)
			{"--seccomp", mustResolveSeccomp(payload.Bwrap, payload.Syscall)},
		},
	); err != nil {
		fmsg.Fatalf("malformed sandbox config: %v", err)
	} else {
		cmd := b.Unwrap()
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		cmd.ExtraFiles = extraFiles

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

func mustResolveSeccomp(bwrap *bwrap.Config, syscall *fst.SyscallConfig) (seccompFd *os.File) {
	if syscall == nil {
		fmsg.VPrintln("syscall filter not configured, PROCEED WITH CAUTION")
		return
	}

	// resolve seccomp filter opts
	var (
		opts    syscallOpts
		optd    []string
		optCond = [...]struct {
			v bool
			o syscallOpts
			d string
		}{
			{!bwrap.UserNS, flagDenyNS, "denyns"},
			{bwrap.NewSession, flagDenyTTY, "denytty"},
			{syscall.DenyDevel, flagDenyDevel, "denydevel"},
			{syscall.Multiarch, flagMultiarch, "multiarch"},
			{syscall.Linux32, flagLinux32, "linux32"},
			{syscall.Can, flagCan, "can"},
			{syscall.Bluetooth, flagBluetooth, "bluetooth"},
		}
	)
	if fmsg.Verbose() {
		optd = make([]string, 1, len(optCond)+1)
		optd[0] = "fortify"
	}
	for _, opt := range optCond {
		if opt.v {
			opts |= opt.o
			if fmsg.Verbose() {
				optd = append(optd, opt.d)
			}
		}
	}
	if fmsg.Verbose() {
		fmsg.VPrintf("seccomp flags: %s", optd)
	}

	// export seccomp filter to tmpfile
	if f, err := tmpfile(); err != nil {
		fmsg.Fatalf("cannot create tmpfile: %v", err)
		panic("unreachable")
	} else {
		mustExportFilter(f, opts)
		seccompFd = f
		return
	}
}

func mustExportFilter(f *os.File, opts syscallOpts) {
	if err := exportFilter(f.Fd(), opts); err != nil {
		fmsg.Fatalf("cannot export seccomp filter: %v", err)
		panic("unreachable")
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		fmsg.Fatalf("cannot lseek seccomp file: %v", err)
		panic("unreachable")
	}
}
