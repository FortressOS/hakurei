package container_test

import (
	"errors"
	"log"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"

	"hakurei.app/container"
	"hakurei.app/internal/hlog"
)

func TestDefaultMsg(t *testing.T) {
	// bypass WrapErr testing behaviour
	t.Setenv("GOPATH", container.Nonexistent)

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

	t.Run("wrapErr", func(t *testing.T) {
		buf := new(strings.Builder)
		log.SetOutput(buf)
		log.SetFlags(0)
		if err := msg.WrapErr(syscall.EBADE, "\x00", "\x00"); err != syscall.EBADE {
			t.Errorf("WrapErr: %v", err)
		}
		msg.PrintBaseErr(syscall.ENOTRECOVERABLE, "cannot cuddle cat:")

		want := "\x00 \x00\ncannot cuddle cat: state not recoverable\n"
		if buf.String() != want {
			t.Errorf("WrapErr: %q, want %q", buf.String(), want)
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

	t.Run("checkedWrappedErr", func(t *testing.T) {
		// temporarily re-enable testing behaviour
		t.Setenv("GOPATH", "")
		wrappedErr := msg.WrapErr(syscall.ENOTRECOVERABLE, "cannot cuddle cat:", syscall.ENOTRECOVERABLE)

		t.Run("string", func(t *testing.T) {
			want := "state not recoverable, a = [cannot cuddle cat: state not recoverable]"
			if got := wrappedErr.Error(); got != want {
				t.Errorf("Error: %q, want %q", got, want)
			}
		})

		t.Run("bad concrete type", func(t *testing.T) {
			if errors.Is(wrappedErr, syscall.ENOTRECOVERABLE) {
				t.Error("incorrect type assertion")
			}
		})
	})
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

func (out *testOutput) WrapErr(err error, a ...any) error       { return hlog.WrapErr(err, a...) }
func (out *testOutput) PrintBaseErr(err error, fallback string) { hlog.PrintBaseError(err, fallback) }

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
