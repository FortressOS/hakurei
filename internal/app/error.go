package app

import (
	"fmt"
	"reflect"
)

// baseError implements a basic error container
type baseError struct {
	Err error
}

func (e *baseError) Error() string {
	return e.Err.Error()
}

func (e *baseError) Unwrap() error {
	return e.Err
}

// BaseError implements an error container with a user-facing message
type BaseError struct {
	message string
	baseError
}

// Message returns a user-facing error message
func (e *BaseError) Message() string {
	return e.message
}

func wrapError(err error, a ...any) *BaseError {
	return &BaseError{
		message:   fmt.Sprintln(a...),
		baseError: baseError{err},
	}
}

var (
	baseErrorType = reflect.TypeFor[*BaseError]()
)

func AsBaseError(err error, target **BaseError) bool {
	v := reflect.ValueOf(err)
	if !v.CanConvert(baseErrorType) {
		return false
	}

	*target = v.Convert(baseErrorType).Interface().(*BaseError)
	return true
}
