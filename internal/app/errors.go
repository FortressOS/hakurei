package app

import (
	"errors"

	"git.gensokyo.uk/security/fortify/internal/fmsg"
)

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
func (e *StateStoreError) save(errs []error) {
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
