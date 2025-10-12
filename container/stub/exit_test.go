package stub_test

import (
	"testing"
	_ "unsafe"

	"hakurei.app/container/stub"
)

//go:linkname handleExitNew hakurei.app/container/stub.handleExitNew
func handleExitNew(_ testing.TB)

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
	t.Parallel()

	t.Run("exit", func(t *testing.T) {
		t.Parallel()
		defer stub.HandleExit(t)
		panic(stub.PanicExit)
	})

	t.Run("goexit", func(t *testing.T) {
		t.Parallel()

		t.Run("FailNow", func(t *testing.T) {
			t.Parallel()

			ot := &overrideTFailNow{T: t}
			defer func() {
				if !ot.failNow {
					t.Errorf("FailNow was never called")
				}
			}()
			defer stub.HandleExit(ot)
			panic(0xcafe0000)
		})

		t.Run("Fail", func(t *testing.T) {
			t.Parallel()

			ot := &overrideTFailNow{T: t}
			defer func() {
				if !ot.fail {
					t.Errorf("Fail was never called")
				}
			}()
			defer handleExitNew(ot)
			panic(0xcafe0000)
		})
	})

	t.Run("nil", func(t *testing.T) {
		t.Parallel()
		defer stub.HandleExit(t)
	})

	t.Run("passthrough", func(t *testing.T) {
		t.Parallel()

		t.Run("toplevel", func(t *testing.T) {
			t.Parallel()

			defer func() {
				want := 0xcafebabe
				if r := recover(); r != want {
					t.Errorf("recover: %v, want %v", r, want)
				}

			}()
			defer stub.HandleExit(t)
			panic(0xcafebabe)
		})

		t.Run("new", func(t *testing.T) {
			t.Parallel()

			defer func() {
				want := 0xcafe
				if r := recover(); r != want {
					t.Errorf("recover: %v, want %v", r, want)
				}

			}()
			defer handleExitNew(t)
			panic(0xcafe)
		})
	})
}
