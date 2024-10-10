package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	"git.ophivana.moe/cat/fortify/dbus"
	"git.ophivana.moe/cat/fortify/internal"
	"git.ophivana.moe/cat/fortify/internal/app"
	"git.ophivana.moe/cat/fortify/internal/shim"
	"git.ophivana.moe/cat/fortify/internal/state"
	"git.ophivana.moe/cat/fortify/internal/verbose"
)

var (
	Version = "impure"
)

func tryVersion() {
	if printVersion {
		fmt.Println(Version)
		os.Exit(0)
	}
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

	// prepare config
	var config *app.Config

	if confPath == "nil" {
		// config from flags
		config = configFromFlags()
	} else {
		// config from file
		if f, err := os.Open(confPath); err != nil {
			fatalf("cannot access config file '%s': %s\n", confPath, err)
		} else {
			if err = json.NewDecoder(f).Decode(&config); err != nil {
				fatalf("cannot parse config file '%s': %s\n", confPath, err)
			}
		}
	}

	// invoke app
	r := 1
	a := app.New()
	if err := a.Seal(config); err != nil {
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

func configFromFlags() (config *app.Config) {
	// initialise config from flags
	config = &app.Config{
		ID:      dbusID,
		User:    userName,
		Command: flag.Args(),
		Method:  launchMethodText,
	}

	// enablements from flags
	if mustWayland {
		config.Confinement.Enablements.Set(state.EnableWayland)
	}
	if mustX {
		config.Confinement.Enablements.Set(state.EnableX)
	}
	if mustDBus {
		config.Confinement.Enablements.Set(state.EnableDBus)
	}
	if mustPulse {
		config.Confinement.Enablements.Set(state.EnablePulse)
	}

	// parse D-Bus config file from flags if applicable
	if mustDBus {
		if dbusConfigSession == "builtin" {
			config.Confinement.SessionBus = dbus.NewConfig(dbusID, true, mpris)
		} else {
			if f, err := os.Open(dbusConfigSession); err != nil {
				fatalf("cannot access session bus proxy config file '%s': %s\n", dbusConfigSession, err)
			} else {
				if err = json.NewDecoder(f).Decode(&config.Confinement.SessionBus); err != nil {
					fatalf("cannot parse session bus proxy config file '%s': %s\n", dbusConfigSession, err)
				}
			}
		}

		// system bus proxy is optional
		if dbusConfigSystem != "nil" {
			if f, err := os.Open(dbusConfigSystem); err != nil {
				fatalf("cannot access system bus proxy config file '%s': %s\n", dbusConfigSystem, err)
			} else {
				if err = json.NewDecoder(f).Decode(&config.Confinement.SystemBus); err != nil {
					fatalf("cannot parse system bus proxy config file '%s': %s\n", dbusConfigSystem, err)
				}
			}
		}

		if dbusVerbose {
			config.Confinement.SessionBus.Log = true
			config.Confinement.SystemBus.Log = true
		}
	}

	return
}

func fatalf(format string, a ...any) {
	fmt.Printf("fortify: "+format, a...)
	os.Exit(1)
}
