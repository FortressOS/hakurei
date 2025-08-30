package container

import (
	"testing"
)

func TestGetSetOutput(t *testing.T) {
	{
		out := GetOutput()
		t.Cleanup(func() { SetOutput(out) })
	}

	t.Run("default", func(t *testing.T) {
		SetOutput(new(stubOutput))
		if v, ok := GetOutput().(*DefaultMsg); ok {
			t.Fatalf("SetOutput: got unexpected output %#v", v)
		}
		SetOutput(nil)
		if _, ok := GetOutput().(*DefaultMsg); !ok {
			t.Fatalf("SetOutput: got unexpected output %#v", GetOutput())
		}
	})

	t.Run("stub", func(t *testing.T) {
		SetOutput(new(stubOutput))
		if _, ok := GetOutput().(*stubOutput); !ok {
			t.Fatalf("SetOutput: got unexpected output %#v", GetOutput())
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
