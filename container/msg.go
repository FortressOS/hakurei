package container

import (
	"log"
	"sync/atomic"
)

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
