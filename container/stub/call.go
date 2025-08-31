package stub

import (
	"slices"
)

// ExpectArgs is an array primarily for storing expected function arguments.
// Its actual use is defined by the implementation.
type ExpectArgs = [5]any

// An Expect stores expected calls of a goroutine.
type Expect struct {
	Calls []Call

	// Tracks are handed out to descendant goroutines in order.
	Tracks []Expect
}

// A Call holds expected arguments of a function call and its outcome.
type Call struct {
	// Name is the function Name of this call. Must be unique.
	Name string
	// Args are the expected arguments of this Call.
	Args ExpectArgs
	// Ret is the return value of this Call.
	Ret any
	// Err is the returned error of this Call.
	Err error
}

// Error returns [Call.Err] if all arguments are true, or [ErrCheck] otherwise.
func (k *Call) Error(ok ...bool) error {
	if !slices.Contains(ok, false) {
		return k.Err
	}
	return ErrCheck
}
