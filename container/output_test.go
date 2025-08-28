package container

import (
	"reflect"
	"syscall"
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

func TestWrapErr(t *testing.T) {
	{
		out := GetOutput()
		t.Cleanup(func() { SetOutput(out) })
	}

	var wrapFp *func(error, ...any) error
	s := new(stubOutput)
	SetOutput(s)
	wrapFp = &s.wrapF

	testCases := []struct {
		name    string
		f       func(t *testing.T)
		wantErr error
		wantA   []any
	}{
		{"suffix nil", func(t *testing.T) {
			if err := wrapErrSuffix(nil, "\x00"); err != nil {
				t.Errorf("wrapErrSuffix: %v", err)
			}
		}, nil, nil},
		{"suffix val", func(t *testing.T) {
			if err := wrapErrSuffix(syscall.ENOTRECOVERABLE, "\x00\x00"); err != syscall.ENOTRECOVERABLE {
				t.Errorf("wrapErrSuffix: %v", err)
			}
		}, syscall.ENOTRECOVERABLE, []any{"\x00\x00", syscall.ENOTRECOVERABLE}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var (
				gotErr error
				gotA   []any
			)
			*wrapFp = func(err error, a ...any) error { gotErr = err; gotA = a; return err }

			tc.f(t)
			if gotErr != tc.wantErr {
				t.Errorf("WrapErr: err = %v, want %v", gotErr, tc.wantErr)
			}

			if !reflect.DeepEqual(gotA, tc.wantA) {
				t.Errorf("WrapErr: a = %v, want %v", gotA, tc.wantA)
			}
		})
	}
}

type stubOutput struct {
	wrapF func(error, ...any) error
}

func (*stubOutput) IsVerbose() bool            { panic("unreachable") }
func (*stubOutput) Verbose(...any)             { panic("unreachable") }
func (*stubOutput) Verbosef(string, ...any)    { panic("unreachable") }
func (*stubOutput) PrintBaseErr(error, string) { panic("unreachable") }
func (*stubOutput) Suspend()                   { panic("unreachable") }
func (*stubOutput) Resume() bool               { panic("unreachable") }
func (*stubOutput) BeforeExit()                { panic("unreachable") }

func (s *stubOutput) WrapErr(err error, v ...any) error {
	if s.wrapF == nil {
		panic("unreachable")
	}
	return s.wrapF(err, v...)
}
