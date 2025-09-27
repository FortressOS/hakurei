package container

import (
	"errors"
	"log"
	"sync/atomic"
)

// MessageError is an error with a user-facing message.
type MessageError interface {
	// Message returns a user-facing error message.
	Message() string

	error
}

// GetErrorMessage returns whether an error implements [MessageError], and the message if it does.
func GetErrorMessage(err error) (string, bool) {
	var e MessageError
	if !errors.As(err, &e) || e == nil {
		return zeroString, false
	}
	return e.Message(), true
}

type Msg interface {
	IsVerbose() bool
	Verbose(v ...any)
	Verbosef(format string, v ...any)

	Suspend()
	Resume() bool
	BeforeExit()
}

type DefaultMsg struct{ inactive atomic.Bool }

func (msg *DefaultMsg) IsVerbose() bool { return true }
func (msg *DefaultMsg) Verbose(v ...any) {
	if !msg.inactive.Load() {
		log.Println(v...)
	}
}
func (msg *DefaultMsg) Verbosef(format string, v ...any) {
	if !msg.inactive.Load() {
		log.Printf(format, v...)
	}
}

func (msg *DefaultMsg) Suspend()     { msg.inactive.Store(true) }
func (msg *DefaultMsg) Resume() bool { return msg.inactive.CompareAndSwap(true, false) }
func (msg *DefaultMsg) BeforeExit()  {}

// msg is the [Msg] implemented used by all exported [container] functions.
var msg Msg = new(DefaultMsg)

// GetOutput returns the current active [Msg] implementation.
func GetOutput() Msg { return msg }

// SetOutput replaces the current active [Msg] implementation.
func SetOutput(v Msg) {
	if v == nil {
		msg = new(DefaultMsg)
	} else {
		msg = v
	}
}
