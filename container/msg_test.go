package container_test

import (
	"errors"
	"log"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"

	"hakurei.app/container"
)

func TestMessageError(t *testing.T) {
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
			got, ok := container.GetErrorMessage(tc.err)
			if got != tc.want {
				t.Errorf("GetErrorMessage: %q, want %q", got, tc.want)
			}
			if ok != tc.wantOk {
				t.Errorf("GetErrorMessage: ok = %v, want %v", ok, tc.wantOk)
			}
		})
	}
}

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

func TestGetSetOutput(t *testing.T) {
	{
		out := container.GetOutput()
		t.Cleanup(func() { container.SetOutput(out) })
	}

	t.Run("default", func(t *testing.T) {
		container.SetOutput(new(stubOutput))
		if v, ok := container.GetOutput().(*container.DefaultMsg); ok {
			t.Fatalf("SetOutput: got unexpected output %#v", v)
		}
		container.SetOutput(nil)
		if _, ok := container.GetOutput().(*container.DefaultMsg); !ok {
			t.Fatalf("SetOutput: got unexpected output %#v", container.GetOutput())
		}
	})

	t.Run("stub", func(t *testing.T) {
		container.SetOutput(new(stubOutput))
		if _, ok := container.GetOutput().(*stubOutput); !ok {
			t.Fatalf("SetOutput: got unexpected output %#v", container.GetOutput())
		}
	})
}

type stubOutput struct {
	wrapF func(error, ...any) error
}

func (*stubOutput) IsVerbose() bool         { panic("unreachable") }
func (*stubOutput) Verbose(...any)          { panic("unreachable") }
func (*stubOutput) Verbosef(string, ...any) { panic("unreachable") }
func (*stubOutput) Suspend()                { panic("unreachable") }
func (*stubOutput) Resume() bool            { panic("unreachable") }
func (*stubOutput) BeforeExit()             { panic("unreachable") }
