package setuid

import (
	"errors"
	"log"

	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
)

func PrintRunStateErr(rs *fst.RunState, runErr error) (code int) {
	code = rs.ExitStatus()

	if runErr != nil {
		if rs.Time == nil {
			fmsg.PrintBaseError(runErr, "cannot start app:")
		} else {
			var e *fmsg.BaseError
			if !fmsg.AsBaseError(runErr, &e) {
				log.Println("wait failed:", runErr)
			} else {
				// Wait only returns either *app.ProcessError or *app.StateStoreError wrapped in a *app.BaseError
				var se *StateStoreError
				if !errors.As(runErr, &se) {
					// does not need special handling
					log.Print(e.Message())
				} else {
					// inner error are either unwrapped store errors
					// or joined errors returned by *appSealTx revert
					// wrapped in *app.BaseError
					var ej RevertCompoundError
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

		if code == 0 {
			code = 126
		}
	}

	if rs.RevertErr != nil {
		var stateStoreError *StateStoreError
		if !errors.As(rs.RevertErr, &stateStoreError) || stateStoreError == nil {
			fmsg.PrintBaseError(rs.RevertErr, "generic fault during cleanup:")
			goto out
		}

		if stateStoreError.Err != nil {
			if len(stateStoreError.Err) == 2 {
				if stateStoreError.Err[0] != nil {
					if joinedErrs, ok := stateStoreError.Err[0].(interface{ Unwrap() []error }); !ok {
						fmsg.PrintBaseError(stateStoreError.Err[0], "generic fault during revert:")
					} else {
						for _, err := range joinedErrs.Unwrap() {
							if err != nil {
								fmsg.PrintBaseError(err, "fault during revert:")
							}
						}
					}
				}
				if stateStoreError.Err[1] != nil {
					log.Printf("cannot close store: %v", stateStoreError.Err[1])
				}
			} else {
				log.Printf("fault during cleanup: %v",
					errors.Join(stateStoreError.Err...))
			}
		}

		if stateStoreError.OpErr != nil {
			log.Printf("blind revert due to store fault: %v",
				stateStoreError.OpErr)
		}

		if stateStoreError.DoErr != nil {
			fmsg.PrintBaseError(stateStoreError.DoErr, "state store operation unsuccessful:")
		}

		if stateStoreError.Inner && stateStoreError.InnerErr != nil {
			fmsg.PrintBaseError(stateStoreError.InnerErr, "cannot destroy state entry:")
		}

	out:
		if code == 0 {
			code = 128
		}
	}
	if rs.WaitErr != nil {
		fmsg.Verbosef("wait: %v", rs.WaitErr)
	}
	return
}

// StateStoreError is returned for a failed state save
type StateStoreError struct {
	// whether inner function was called
	Inner bool
	// returned by the Save/Destroy method of [state.Cursor]
	InnerErr error
	// returned by the Do method of [state.Store]
	DoErr error
	// stores an arbitrary store operation error
	OpErr error
	// stores arbitrary errors
	Err []error
}

// save saves arbitrary errors in [StateStoreError] once.
func (e *StateStoreError) save(errs ...error) {
	if len(errs) == 0 || e.Err != nil {
		panic("invalid call to save")
	}
	e.Err = errs
}

func (e *StateStoreError) equiv(a ...any) error {
	if e.Inner && e.InnerErr == nil && e.DoErr == nil && e.OpErr == nil && errors.Join(e.Err...) == nil {
		return nil
	} else {
		return fmsg.WrapErrorSuffix(e, a...)
	}
}

func (e *StateStoreError) Error() string {
	if e.Inner && e.InnerErr != nil {
		return e.InnerErr.Error()
	}
	if e.DoErr != nil {
		return e.DoErr.Error()
	}
	if e.OpErr != nil {
		return e.OpErr.Error()
	}
	if err := errors.Join(e.Err...); err != nil {
		return err.Error()
	}

	// equiv nullifies e for values where this is reached
	panic("unreachable")
}

func (e *StateStoreError) Unwrap() (errs []error) {
	errs = make([]error, 0, 3)
	if e.InnerErr != nil {
		errs = append(errs, e.InnerErr)
	}
	if e.DoErr != nil {
		errs = append(errs, e.DoErr)
	}
	if e.OpErr != nil {
		errs = append(errs, e.OpErr)
	}
	if err := errors.Join(e.Err...); err != nil {
		errs = append(errs, err)
	}
	return
}

// A RevertCompoundError encapsulates errors returned by
// the Revert method of [system.I].
type RevertCompoundError interface {
	Error() string
	Unwrap() []error
}
