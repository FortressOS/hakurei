package container

import (
	"log"
	"sync/atomic"
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

func (msg *DefaultMsg) WrapErr(err error, a ...any) error {
	log.Println(a...)
	return err
}
func (msg *DefaultMsg) PrintBaseErr(err error, fallback string) { log.Println(fallback, err) }

func (msg *DefaultMsg) Suspend()     { msg.inactive.Store(true) }
func (msg *DefaultMsg) Resume() bool { return msg.inactive.CompareAndSwap(true, false) }
func (msg *DefaultMsg) BeforeExit()  {}
