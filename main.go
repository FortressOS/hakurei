package main

import (
	"errors"
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

	// version/license command early exit
	tryVersion()
	tryLicense()

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
		r = 1

		var e *app.BaseError
		if !app.AsBaseError(err, &e) {
			fmt.Println("fortify: wait failed:", err)
		} else {
			// Wait only returns either *app.ProcessError or *app.StateStoreError wrapped in a *app.BaseError
			var se *app.StateStoreError
			if !errors.As(err, &se) {
				// does not need special handling
				fmt.Print("fortify: " + e.Message())
			} else {
				// inner error are either unwrapped store errors
				// or joined errors returned by *appSealTx revert
				// wrapped in *app.BaseError
				var ej app.RevertCompoundError
				if !errors.As(se.InnerErr, &ej) {
					// does not require special handling
					fmt.Print("fortify: " + e.Message())
				} else {
					errs := ej.Unwrap()

					// every error here is wrapped in *app.BaseError
					for _, ei := range errs {
						var eb *app.BaseError
						if !errors.As(ei, &eb) {
							// unreachable
							fmt.Println("fortify: invalid error type returned by revert:", ei)
						} else {
							// print inner *app.BaseError message
							fmt.Print("fortify: " + eb.Message())
						}
					}
				}
			}
		}
	}
	if err := a.WaitErr(); err != nil {
		fmt.Println("fortify: inner wait failed:", err)
	}
	os.Exit(r)
}

func logBaseError(err error, message string) {
	var e *app.BaseError

	if app.AsBaseError(err, &e) {
		fmt.Print("fortify: " + e.Message())
	} else {
		fmt.Println(message, err)
	}
}

func fatalf(format string, a ...any) {
	fmt.Printf("fortify: "+format, a...)
	os.Exit(1)
}
