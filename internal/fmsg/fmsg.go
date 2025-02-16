// Package fmsg provides various functions for output messages.
package fmsg

import (
	"bytes"
	"io"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
)

const (
	bufSize    = 4 * 1024
	bufSizeMax = 16 * 1024 * 1024
)

var o = &suspendable{w: os.Stderr}

// Prepare configures the system logger for [Suspend] and [Resume] to take effect.
func Prepare(prefix string) { log.SetPrefix(prefix + ": "); log.SetFlags(0); log.SetOutput(o) }

type suspendable struct {
	w io.Writer
	s atomic.Bool

	buf     bytes.Buffer
	bufOnce sync.Once
	bufMu   sync.Mutex
	dropped int
}

func (s *suspendable) Write(p []byte) (n int, err error) {
	if !s.s.Load() {
		return s.w.Write(p)
	}
	s.bufOnce.Do(func() { s.prepareBuf() })

	s.bufMu.Lock()
	defer s.bufMu.Unlock()

	if l := len(p); s.buf.Len()+l > bufSizeMax {
		s.dropped += l
		return 0, syscall.ENOMEM
	}
	return s.buf.Write(p)
}

func (s *suspendable) prepareBuf()   { s.buf.Grow(bufSize) }
func (s *suspendable) Suspend() bool { return o.s.CompareAndSwap(false, true) }
func (s *suspendable) Resume() (resumed bool, dropped uintptr, n int64, err error) {
	if o.s.CompareAndSwap(true, false) {
		o.bufMu.Lock()
		defer o.bufMu.Unlock()

		resumed = true
		dropped = uintptr(o.dropped)

		o.dropped = 0
		n, err = io.Copy(s.w, &s.buf)
		s.buf = bytes.Buffer{}
		s.prepareBuf()
	}
	return
}

func Suspend() bool { return o.Suspend() }
func Resume() bool {
	resumed, dropped, _, err := o.Resume()
	if err != nil {
		// probably going to result in an error as well,
		// so this call is as good as unreachable
		log.Printf("cannot dump buffer on resume: %v", err)
	}
	if resumed && dropped > 0 {
		log.Fatalf("dropped %d bytes while output is suspended", dropped)
	}
	return resumed
}

func BeforeExit() {
	if Resume() {
		log.Printf("beforeExit reached on suspended output")
	}
}
