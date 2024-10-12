package main

import (
	"flag"
	"fmt"
	"os"

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

	if internal.SdBootedV {
		verbose.Println("system booted with systemd as init system")
	}

	// launcher payload early exit
	if printVersion && printLicense {
		shim.Try()
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
