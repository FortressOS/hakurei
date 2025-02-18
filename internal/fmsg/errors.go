package fmsg

import (
	"fmt"
	"log"
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

// WrapError wraps an error with a corresponding message.
func WrapError(err error, a ...any) error {
	if err == nil {
		return nil
	}
	return wrapError(err, fmt.Sprintln(a...))
}

// WrapErrorSuffix wraps an error with a corresponding message with err at the end of the message.
func WrapErrorSuffix(err error, a ...any) error {
	if err == nil {
		return nil
	}
	return wrapError(err, fmt.Sprintln(append(a, err)...))
}

// WrapErrorFunc wraps an error with a corresponding message returned by f.
func WrapErrorFunc(err error, f func(err error) string) error {
	if err == nil {
		return nil
	}
	return wrapError(err, f(err))
}

func wrapError(err error, message string) *BaseError {
	return &BaseError{message, baseError{err}}
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

func PrintBaseError(err error, fallback string) {
	var e *BaseError

	if AsBaseError(err, &e) {
		log.Print(e.Message())
	} else {
		log.Println(fallback, err)
	}
}
