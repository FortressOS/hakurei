package app

import (
	"errors"
	"log"

	"hakurei.app/container"
	"hakurei.app/internal/hlog"
)

// PrintRunStateErr prints an error message via [log] if runErr is not nil, and returns an appropriate exit code.
//
// TODO(ophestra): remove this function once RunState has been replaced
func PrintRunStateErr(rs *RunState, runErr error) (code int) {
	code = rs.ExitStatus()

	if runErr != nil {
		if rs.Time == nil {
			// no process has been created
			printMessageError("cannot start app:", runErr)
		} else {
			if m, ok := container.GetErrorMessage(runErr); !ok {
				// catch-all for unexpected errors
				log.Println("run returned error:", runErr)
			} else {
				var se *StateStoreError
				if !errors.As(runErr, &se) {
					// this could only be returned from a shim setup failure path
					log.Print(m)
				} else {
					// InnerErr is returned by c.Save(&sd, seal.ct), and are always unwrapped
					printMessageError("error returned during revert:",
						&FinaliseError{Step: "save process state", Err: se.InnerErr})
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
			printMessageError("cannot clean up:", rs.RevertErr)
			goto out
		}

		if stateStoreError.Errs != nil {
			if len(stateStoreError.Errs) == 2 { // storeErr.save(revertErr, store.Close())
				if stateStoreError.Errs[0] != nil { // revertErr is MessageError joined by errors.Join
					var joinedErrors interface {
						Unwrap() []error
						error
					}
					if !errors.As(stateStoreError.Errs[0], &joinedErrors) {
						printMessageError("cannot revert:", stateStoreError.Errs[0])
					} else {
						for _, err := range joinedErrors.Unwrap() {
							if err != nil {
								printMessageError("cannot revert:", err)
							}
						}
					}
				}
				if stateStoreError.Errs[1] != nil { // store.Close() is joined by errors.Join
					log.Printf("cannot close store: %v", stateStoreError.Errs[1])
				}
			} else {
				log.Printf("fault during cleanup: %v", errors.Join(stateStoreError.Errs...))
			}
		}

		if stateStoreError.OpErr != nil {
			log.Printf("blind revert due to store fault: %v", stateStoreError.OpErr)
		}

		if stateStoreError.DoErr != nil {
			printMessageError("state store operation unsuccessful:", stateStoreError.DoErr)
		}

		if stateStoreError.Inner && stateStoreError.InnerErr != nil {
			printMessageError("cannot destroy state entry:", stateStoreError.InnerErr)
		}

	out:
		if code == 0 {
			code = 128
		}
	}
	if rs.WaitErr != nil {
		hlog.Verbosef("wait: %v", rs.WaitErr)
	}
	return
}

// TODO(ophestra): this duplicates code in cmd/hakurei/command.go, keep this up to date until removal
func printMessageError(fallback string, err error) {
	if m, ok := container.GetErrorMessage(err); ok {
		if m != "\x00" {
			log.Print(m)
		}
	} else {
		log.Println(fallback, err)
	}
}

// StateStoreError is returned for a failed state save.
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
	Errs []error
}

// save saves arbitrary errors in [StateStoreError.Errs] once.
func (e *StateStoreError) save(errs ...error) {
	if len(errs) == 0 || e.Errs != nil {
		panic("invalid call to save")
	}
	e.Errs = errs
}

// equiv returns an error that [StateStoreError] is equivalent to, including nil.
func (e *StateStoreError) equiv(step string) error {
	if e.Inner && e.InnerErr == nil && e.DoErr == nil && e.OpErr == nil && errors.Join(e.Errs...) == nil {
		return nil
	} else {
		return &FinaliseError{Step: step, Err: e}
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
	if err := errors.Join(e.Errs...); err != nil {
		return err.Error()
	}

	// equiv nullifies e for values where this is reached
	panic("unreachable")
}

func (e *StateStoreError) Unwrap() (errs []error) {
	errs = make([]error, 0, 3+len(e.Errs))
	if e.InnerErr != nil {
		errs = append(errs, e.InnerErr)
	}
	if e.DoErr != nil {
		errs = append(errs, e.DoErr)
	}
	if e.OpErr != nil {
		errs = append(errs, e.OpErr)
	}
	for _, err := range e.Errs {
		if err != nil {
			errs = append(errs, err)
		}
	}
	return
}
