package stub_test

import (
	"testing"

	"hakurei.app/container/stub"
)

func TestHandleExit(t *testing.T) {
	t.Run("exit", func(t *testing.T) {
		defer stub.HandleExit()
		panic(stub.PanicExit)
	})

	t.Run("nil", func(t *testing.T) {
		defer stub.HandleExit()
	})

	t.Run("passthrough", func(t *testing.T) {
		defer func() {
			want := 0xcafebabe
			if r := recover(); r != want {
				t.Errorf("recover: %v, want %v", r, want)
			}

		}()
		defer stub.HandleExit()
		panic(0xcafebabe)
	})
}
