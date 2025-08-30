package system

import (
	"errors"
	"net"
	"os"

	"hakurei.app/container"
)

var msg container.Msg = new(container.DefaultMsg)

func SetOutput(v container.Msg) {
	if v == nil {
		msg = new(container.DefaultMsg)
	} else {
		msg = v
	}
}

// OpError is returned by [I.Commit] and [I.Revert].
type OpError struct {
	Op      string
	Err     error
	Message string
	Revert  bool
}

func (e *OpError) Unwrap() error { return e.Err }
func (e *OpError) Error() string {
	if e.Message != "" {
		return e.Message
	}

	switch {
	case errors.As(e.Err, new(*os.PathError)), errors.As(e.Err, new(*net.OpError)):
		return e.Err.Error()

	default:
		if !e.Revert {
			return "cannot apply " + e.Op + ": " + e.Err.Error()
		} else {
			return "cannot revert " + e.Op + ": " + e.Err.Error()
		}
	}
}

// newOpError returns an [OpError] without a message string.
func newOpError(op string, err error, revert bool) error {
	if err == nil {
		return nil
	}
	return &OpError{op, err, "", revert}
}

// newOpErrorMessage returns an [OpError] with an overriding message string.
func newOpErrorMessage(op string, err error, message string, revert bool) error {
	if err == nil {
		return nil
	}
	return &OpError{op, err, message, revert}
}

func printJoinedError(println func(v ...any), fallback string, err error) {
	var joinErr interface {
		Unwrap() []error
		error
	}
	if !errors.As(err, &joinErr) {
		println(fallback, err)
	} else {
		for _, err = range joinErr.Unwrap() {
			println(err.Error())
		}
	}
}
