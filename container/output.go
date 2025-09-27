package container

import (
	"bytes"
	"io"
	"sync"
	"sync/atomic"
	"syscall"
)

const (
	suspendBufInitial = 1 << 12
	suspendBufMax     = 1 << 24
)

// Suspendable proxies writes to a downstream [io.Writer] but optionally withholds writes
// between calls to Suspend and Resume.
type Suspendable struct {
	Downstream io.Writer

	s atomic.Bool

	buf bytes.Buffer
	// for growing buf
	bufOnce sync.Once
	// for synchronising all other buf operations
	bufMu sync.Mutex

	dropped int
}

func (s *Suspendable) Write(p []byte) (n int, err error) {
	if !s.s.Load() {
		return s.Downstream.Write(p)
	}
	s.bufOnce.Do(func() { s.buf.Grow(suspendBufInitial) })

	s.bufMu.Lock()
	defer s.bufMu.Unlock()

	if free := suspendBufMax - s.buf.Len(); free < len(p) {
		// fast path
		if free <= 0 {
			s.dropped += len(p)
			return 0, syscall.ENOMEM
		}

		n, _ = s.buf.Write(p[:free])
		err = syscall.ENOMEM
		s.dropped += len(p) - n
		return
	}

	return s.buf.Write(p)
}

// IsSuspended returns whether [Suspendable] is currently between a call to Suspend and Resume.
func (s *Suspendable) IsSuspended() bool { return s.s.Load() }

// Suspend causes [Suspendable] to start withholding output in its buffer.
func (s *Suspendable) Suspend() bool { return s.s.CompareAndSwap(false, true) }

// Resume undoes the effect of Suspend and dumps the buffered into the downstream [io.Writer].
func (s *Suspendable) Resume() (resumed bool, dropped uintptr, n int64, err error) {
	if s.s.CompareAndSwap(true, false) {
		s.bufMu.Lock()
		defer s.bufMu.Unlock()

		resumed = true
		dropped = uintptr(s.dropped)

		s.dropped = 0
		n, err = io.Copy(s.Downstream, &s.buf)
		s.buf.Reset()
	}
	return
}

var msg Msg = new(DefaultMsg)

func GetOutput() Msg { return msg }
func SetOutput(v Msg) {
	if v == nil {
		msg = new(DefaultMsg)
	} else {
		msg = v
	}
}
