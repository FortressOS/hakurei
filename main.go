package main

import (
	"flag"

	"git.ophivana.moe/security/fortify/internal"
	"git.ophivana.moe/security/fortify/internal/app"
	"git.ophivana.moe/security/fortify/internal/fmsg"
	"git.ophivana.moe/security/fortify/internal/linux"
)

var (
	flagVerbose bool
)

func init() {
	flag.BoolVar(&flagVerbose, "v", false, "Verbose output")
}

var os = new(linux.Std)

func main() {
	if err := internal.PR_SET_DUMPABLE__SUID_DUMP_DISABLE(); err != nil {
		fmsg.Printf("cannot set SUID_DUMP_DISABLE: %s", err)
		// not fatal: this program runs as the privileged user
	}

	flag.Parse()
	fmsg.SetVerbose(flagVerbose)

	if os.SdBooted() {
		fmsg.VPrintln("system booted with systemd as init system")
	}

	// root check
	if os.Geteuid() == 0 {
		fmsg.Fatal("this program must not run as root")
		panic("unreachable")
	}

	// version/license/template command early exit
	tryVersion()
	tryLicense()
	tryTemplate()

	// state query command early exit
	tryState()

	// invoke app
	a, err := app.New(os)
	if err != nil {
		fmsg.Fatalf("cannot create app: %s\n", err)
	} else if err = a.Seal(loadConfig()); err != nil {
		logBaseError(err, "cannot seal app:")
		fmsg.Exit(1)
	} else if err = a.Start(); err != nil {
		logBaseError(err, "cannot start app:")
	}

	var r int
	// wait must be called regardless of result of start
	if r, err = a.Wait(); err != nil {
		if r < 1 {
			r = 1
		}
		logWaitError(err)
	}
	if err = a.WaitErr(); err != nil {
		fmsg.Println("inner wait failed:", err)
	}
	fmsg.Exit(r)
}
