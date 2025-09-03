package stub_test

import (
	"testing"
	_ "unsafe"

	"hakurei.app/container/stub"
)

//go:linkname handleExit hakurei.app/container/stub.handleExit
func handleExit(_ testing.TB, _ bool)

// overrideTFailNow overrides the Fail and FailNow method.
type overrideTFailNow struct {
	*testing.T
	failNow bool
	fail    bool
}

func (o *overrideTFailNow) FailNow() {
	if o.failNow {
		o.Errorf("attempted to FailNow twice")
	}
	o.failNow = true
}

func (o *overrideTFailNow) Fail() {
	if o.fail {
		o.Errorf("attempted to Fail twice")
	}
	o.fail = true
}

func TestHandleExit(t *testing.T) {
	t.Run("exit", func(t *testing.T) {
		defer handleExit(t, true)
		panic(stub.PanicExit)
	})

	t.Run("goexit", func(t *testing.T) {
		t.Run("FailNow", func(t *testing.T) {
			ot := &overrideTFailNow{T: t}
			defer func() {
				if !ot.failNow {
					t.Errorf("FailNow was never called")
				}
			}()
			defer handleExit(ot, true)
			panic(0xcafe0000)
		})

		t.Run("Fail", func(t *testing.T) {
			ot := &overrideTFailNow{T: t}
			defer func() {
				if !ot.fail {
					t.Errorf("Fail was never called")
				}
			}()
			defer handleExit(ot, false)
			panic(0xcafe0000)
		})
	})

	t.Run("nil", func(t *testing.T) {
		defer handleExit(t, true)
	})

	t.Run("passthrough", func(t *testing.T) {
		defer func() {
			want := 0xcafebabe
			if r := recover(); r != want {
				t.Errorf("recover: %v, want %v", r, want)
			}

		}()
		defer handleExit(t, true)
		panic(0xcafebabe)
	})
}
