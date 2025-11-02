package message_test

import (
	"bytes"
	"errors"
	"io"
	"log"
	"strings"
	"syscall"
	"testing"

	"hakurei.app/container"
	"hakurei.app/container/stub"
	"hakurei.app/message"
)

func TestMessageError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		err    error
		want   string
		wantOk bool
	}{
		{"nil", nil, "", false},
		{"new", errors.New(":3"), "", false},
		{"start", &container.StartError{
			Step: "meow",
			Err:  syscall.ENOTRECOVERABLE,
		}, "cannot meow: state not recoverable", true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := message.GetMessage(tc.err)
			if got != tc.want {
				t.Errorf("GetMessage: %q, want %q", got, tc.want)
			}
			if ok != tc.wantOk {
				t.Errorf("GetMessage: ok = %v, want %v", ok, tc.wantOk)
			}
		})
	}
}

func TestDefaultMsg(t *testing.T) {
	// copied from output.go
	const suspendBufMax = 1 << 24
	t.Parallel()

	t.Run("logger", func(t *testing.T) {
		t.Run("nil", func(t *testing.T) {
			got := message.New(nil).GetLogger()

			if out := got.Writer().(*message.Suspendable).Downstream; out != log.Writer() {
				t.Errorf("GetLogger: Downstream = %#v", out)
			}

			if prefix := got.Prefix(); prefix != "container: " {
				t.Errorf("GetLogger: prefix = %q", prefix)
			}
		})

		t.Run("takeover", func(t *testing.T) {
			l := log.New(io.Discard, "\x00", 0xdeadbeef)
			got := message.New(l)

			if logger := got.GetLogger(); logger != l {
				t.Errorf("GetLogger: %#v, want %#v", logger, l)
			}

			if ds := l.Writer().(*message.Suspendable).Downstream; ds != io.Discard {
				t.Errorf("GetLogger: Downstream = %#v", ds)
			}
		})
	})

	dw := expectWriter{t: t}

	steps := []struct {
		name     string
		pt, next []byte
		err      error

		f func(t *testing.T, msg message.Msg)
	}{
		{"zero verbose", nil, nil, nil, func(t *testing.T, msg message.Msg) {
			if msg.IsVerbose() {
				t.Error("IsVerbose unexpected true")
			}
		}},

		{"swap false", nil, nil, nil, func(t *testing.T, msg message.Msg) {
			if msg.SwapVerbose(false) {
				t.Error("SwapVerbose unexpected true")
			}
		}},
		{"write discard", nil, nil, nil, func(_ *testing.T, msg message.Msg) {
			msg.Verbose("\x00")
			msg.Verbosef("\x00")
		}},
		{"verbose false", nil, nil, nil, func(t *testing.T, msg message.Msg) {
			if msg.IsVerbose() {
				t.Error("IsVerbose unexpected true")
			}
		}},

		{"swap true", nil, nil, nil, func(t *testing.T, msg message.Msg) {
			if msg.SwapVerbose(true) {
				t.Error("SwapVerbose unexpected true")
			}
		}},
		{"write verbose", []byte("test: \x00\n"), nil, nil, func(_ *testing.T, msg message.Msg) {
			msg.Verbose("\x00")
		}},
		{"write verbosef", []byte(`test: "\x00"` + "\n"), nil, nil, func(_ *testing.T, msg message.Msg) {
			msg.Verbosef("%q", "\x00")
		}},
		{"verbose true", nil, nil, nil, func(t *testing.T, msg message.Msg) {
			if !msg.IsVerbose() {
				t.Error("IsVerbose unexpected false")
			}
		}},

		{"resume noop", nil, nil, nil, func(t *testing.T, msg message.Msg) {
			if msg.Resume() {
				t.Error("Resume unexpected success")
			}
		}},
		{"beforeExit noop", nil, nil, nil, func(_ *testing.T, msg message.Msg) {
			msg.BeforeExit()
		}},

		{"beforeExit suspend", nil, nil, nil, func(_ *testing.T, msg message.Msg) {
			msg.Suspend()
		}},
		{"beforeExit message", []byte("test: beforeExit reached on suspended output\n"), nil, nil, func(_ *testing.T, msg message.Msg) {
			msg.BeforeExit()
		}},
		{"post beforeExit resume noop", nil, nil, nil, func(t *testing.T, msg message.Msg) {
			if msg.Resume() {
				t.Error("Resume unexpected success")
			}
		}},

		{"suspend", nil, nil, nil, func(_ *testing.T, msg message.Msg) {
			msg.Suspend()
		}},
		{"suspend write", nil, nil, nil, func(_ *testing.T, msg message.Msg) {
			msg.GetLogger().Print("\x00")
		}},
		{"resume error", []byte("test: \x00\n"), []byte("test: cannot dump buffer on resume: unique error 0 injected by the test suite\n"), stub.UniqueError(0), func(t *testing.T, msg message.Msg) {
			if !msg.Resume() {
				t.Error("Resume unexpected failure")
			}
		}},

		{"suspend drop", nil, nil, nil, func(_ *testing.T, msg message.Msg) {
			msg.Suspend()
		}},
		{"suspend write fill", nil, nil, nil, func(_ *testing.T, msg message.Msg) {
			msg.GetLogger().Print(strings.Repeat("\x00", suspendBufMax))
		}},
		{"resume dropped", append([]byte("test: "), bytes.Repeat([]byte{0}, suspendBufMax-6)...), []byte("test: dropped 7 bytes while output is suspended\n"), nil, func(t *testing.T, msg message.Msg) {
			if !msg.Resume() {
				t.Error("Resume unexpected failure")
			}
		}},
	}

	msg := message.New(log.New(&dw, "test: ", 0))
	for _, step := range steps {
		// these share the same writer, so cannot be subtests
		t.Logf("running step %q", step.name)
		dw.expect, dw.next, dw.err = step.pt, step.next, step.err
		step.f(t, msg)
		if dw.expect != nil {
			t.Errorf("expect: %q", string(dw.expect))
		}
	}
}
