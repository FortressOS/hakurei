package main

import (
	"flag"
	"fmt"
	"os"
	"syscall"

	"git.ophivana.moe/cat/fortify/internal"
	"git.ophivana.moe/cat/fortify/internal/app"
	"git.ophivana.moe/cat/fortify/internal/shim"
	"git.ophivana.moe/cat/fortify/internal/verbose"
)

var (
	flagVerbose bool
)

func init() {
	flag.BoolVar(&flagVerbose, "v", false, "Verbose output")
}

func main() {
	flag.Parse()
	verbose.Set(flagVerbose)

	// linux/sched/coredump.h
	if _, _, errno := syscall.RawSyscall(syscall.SYS_PRCTL, syscall.PR_SET_DUMPABLE, 0, 0); errno != 0 {
		fmt.Printf("fortify: cannot set SUID_DUMP_DISABLE: %s", errno.Error())
	} else {
		verbose.Println("prctl(PR_SET_DUMPABLE, SUID_DUMP_DISABLE) succeeded")
	}

	if internal.SdBootedV {
		verbose.Println("system booted with systemd as init system")
	}

	// shim early exit
	shim.Try()

	// root check
	if os.Getuid() == 0 {
		fmt.Println("fortify: this program must not run as root")
		os.Exit(1)
	}

	// version/license/template command early exit
	tryVersion()
	tryLicense()
	tryTemplate()

	// state query command early exit
	tryState()

	// invoke app
	r := 1
	a := app.New()
	if err := a.Seal(loadConfig()); err != nil {
		logBaseError(err, "fortify: cannot seal app:")
	} else if err = a.Start(); err != nil {
		logBaseError(err, "fortify: cannot start app:")
	} else if r, err = a.Wait(); err != nil {
		if r < 1 {
			r = 1
		}
		logWaitError(err)
	}
	if err := a.WaitErr(); err != nil {
		fmt.Println("fortify: inner wait failed:", err)
	}
	os.Exit(r)
}
