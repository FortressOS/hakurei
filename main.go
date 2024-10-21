package main

import (
	"flag"
	"os"
	"syscall"

	"git.ophivana.moe/security/fortify/internal"
	"git.ophivana.moe/security/fortify/internal/app"
	"git.ophivana.moe/security/fortify/internal/fmsg"
	init0 "git.ophivana.moe/security/fortify/internal/init"
	"git.ophivana.moe/security/fortify/internal/shim"
)

var (
	flagVerbose bool
)

func init() {
	flag.BoolVar(&flagVerbose, "v", false, "Verbose output")
}

func main() {
	// linux/sched/coredump.h
	if _, _, errno := syscall.RawSyscall(syscall.SYS_PRCTL, syscall.PR_SET_DUMPABLE, 0, 0); errno != 0 {
		fmsg.Printf("fortify: cannot set SUID_DUMP_DISABLE: %s", errno.Error())
	}

	flag.Parse()
	fmsg.SetVerbose(flagVerbose)

	if internal.SdBootedV {
		fmsg.VPrintln("system booted with systemd as init system")
	}

	// shim/init early exit
	init0.Try()
	shim.Try()

	// root check
	if os.Getuid() == 0 {
		fmsg.Println("this program must not run as root")
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
	a, err := app.New()
	if err != nil {
		fmsg.Fatalf("cannot create app: %s\n", err)
	} else if err = a.Seal(loadConfig()); err != nil {
		logBaseError(err, "fortify: cannot seal app:")
	} else if err = a.Start(); err != nil {
		logBaseError(err, "fortify: cannot start app:")
	} else if r, err = a.Wait(); err != nil {
		if r < 1 {
			r = 1
		}
		logWaitError(err)
	}
	if err = a.WaitErr(); err != nil {
		fmsg.Println("inner wait failed:", err)
	}
	os.Exit(r)
}
