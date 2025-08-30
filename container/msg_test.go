package container_test

import (
	"log"
	"strings"
	"sync/atomic"
	"testing"

	"hakurei.app/container"
)

func TestDefaultMsg(t *testing.T) {
	{
		w := log.Writer()
		f := log.Flags()
		t.Cleanup(func() { log.SetOutput(w); log.SetFlags(f) })
	}
	msg := new(container.DefaultMsg)

	t.Run("is verbose", func(t *testing.T) {
		if !msg.IsVerbose() {
			t.Error("IsVerbose unexpected outcome")
		}
	})

	t.Run("verbose", func(t *testing.T) {
		log.SetOutput(panicWriter{})
		msg.Suspend()
		msg.Verbose()
		msg.Verbosef("\x00")
		msg.Resume()

		buf := new(strings.Builder)
		log.SetOutput(buf)
		log.SetFlags(0)
		msg.Verbose()
		msg.Verbosef("\x00")

		want := "\n\x00\n"
		if buf.String() != want {
			t.Errorf("Verbose: %q, want %q", buf.String(), want)
		}
	})

	t.Run("inactive", func(t *testing.T) {
		{
			inactive := msg.Resume()
			if inactive {
				t.Cleanup(func() { msg.Suspend() })
			}
		}

		if msg.Resume() {
			t.Error("Resume unexpected outcome")
		}

		msg.Suspend()
		if !msg.Resume() {
			t.Error("Resume unexpected outcome")
		}
	})

	// the function is a noop
	t.Run("beforeExit", func(t *testing.T) { msg.BeforeExit() })
}

type panicWriter struct{}

func (panicWriter) Write([]byte) (int, error) { panic("unreachable") }

func saveRestoreOutput(t *testing.T) {
	out := container.GetOutput()
	t.Cleanup(func() { container.SetOutput(out) })
}

func replaceOutput(t *testing.T) {
	saveRestoreOutput(t)
	container.SetOutput(&testOutput{t: t})
}

type testOutput struct {
	t         *testing.T
	suspended atomic.Bool
}

func (out *testOutput) IsVerbose() bool { return testing.Verbose() }

func (out *testOutput) Verbose(v ...any) {
	if !out.IsVerbose() {
		return
	}
	out.t.Log(v...)
}

func (out *testOutput) Verbosef(format string, v ...any) {
	if !out.IsVerbose() {
		return
	}
	out.t.Logf(format, v...)
}

func (out *testOutput) Suspend() {
	if out.suspended.CompareAndSwap(false, true) {
		out.Verbose("suspend called")
		return
	}
	out.Verbose("suspend called on suspended output")
}

func (out *testOutput) Resume() bool {
	if out.suspended.CompareAndSwap(true, false) {
		out.Verbose("resume called")
		return true
	}
	out.Verbose("resume called on unsuspended output")
	return false
}

func (out *testOutput) BeforeExit() { out.Verbose("beforeExit called") }
