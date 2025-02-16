package main

import (
	"errors"
	"log"

	"git.gensokyo.uk/security/fortify/internal/app"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
)

func logWaitError(err error) {
	var e *fmsg.BaseError
	if !fmsg.AsBaseError(err, &e) {
		log.Println("wait failed:", err)
	} else {
		// Wait only returns either *app.ProcessError or *app.StateStoreError wrapped in a *app.BaseError
		var se *app.StateStoreError
		if !errors.As(err, &se) {
			// does not need special handling
			log.Print(e.Message())
		} else {
			// inner error are either unwrapped store errors
			// or joined errors returned by *appSealTx revert
			// wrapped in *app.BaseError
			var ej app.RevertCompoundError
			if !errors.As(se.InnerErr, &ej) {
				// does not require special handling
				log.Print(e.Message())
			} else {
				errs := ej.Unwrap()

				// every error here is wrapped in *app.BaseError
				for _, ei := range errs {
					var eb *fmsg.BaseError
					if !errors.As(ei, &eb) {
						// unreachable
						log.Println("invalid error type returned by revert:", ei)
					} else {
						// print inner *app.BaseError message
						log.Print(eb.Message())
					}
				}
			}
		}
	}
}

func logBaseError(err error, message string) {
	var e *fmsg.BaseError

	if fmsg.AsBaseError(err, &e) {
		log.Print(e.Message())
	} else {
		log.Println(message, err)
	}
}
