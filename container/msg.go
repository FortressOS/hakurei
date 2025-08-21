package container

import (
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"sync/atomic"
	"testing"
)

type Msg interface {
	IsVerbose() bool
	Verbose(v ...any)
	Verbosef(format string, v ...any)
	WrapErr(err error, a ...any) error
	PrintBaseErr(err error, fallback string)

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

// checkedWrappedErr implements error with strict checks for wrapped values.
type checkedWrappedErr struct {
	err error
	a   []any
}

func (c *checkedWrappedErr) Error() string { return fmt.Sprintf("%v, a = %s", c.err, c.a) }
func (c *checkedWrappedErr) Is(err error) bool {
	var concreteErr *checkedWrappedErr
	if !errors.As(err, &concreteErr) {
		return false
	}
	return reflect.DeepEqual(c, concreteErr)
}

func (msg *DefaultMsg) WrapErr(err error, a ...any) error {
	// provide a mostly bulletproof path to bypass this behaviour in tests
	if testing.Testing() && os.Getenv("GOPATH") != Nonexistent {
		return &checkedWrappedErr{err, a}
	}

	log.Println(a...)
	return err
}
func (msg *DefaultMsg) PrintBaseErr(err error, fallback string) { log.Println(fallback, err) }

func (msg *DefaultMsg) Suspend()     { msg.inactive.Store(true) }
func (msg *DefaultMsg) Resume() bool { return msg.inactive.CompareAndSwap(true, false) }
func (msg *DefaultMsg) BeforeExit()  {}
