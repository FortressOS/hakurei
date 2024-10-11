package main

import (
	"errors"
	"fmt"
	"os"

	"git.ophivana.moe/cat/fortify/internal/app"
)

func logWaitError(err error) {
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
