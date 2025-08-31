package stub_test

import (
	"errors"
	"syscall"
	"testing"

	"hakurei.app/container/stub"
)

func TestUniqueError(t *testing.T) {
	t.Run("format", func(t *testing.T) {
		want := "unique error 2989 injected by the test suite"
		if got := stub.UniqueError(0xbad).Error(); got != want {
			t.Errorf("Error: %q, want %q", got, want)
		}
	})

	t.Run("is", func(t *testing.T) {
		t.Run("type", func(t *testing.T) {
			if errors.Is(stub.UniqueError(0), syscall.ENOTRECOVERABLE) {
				t.Error("Is: unexpected true")
			}
		})

		t.Run("val", func(t *testing.T) {
			if errors.Is(stub.UniqueError(0), stub.UniqueError(1)) {
				t.Error("Is: unexpected true")
			}
			if !errors.Is(stub.UniqueError(0xbad), stub.UniqueError(0xbad)) {
				t.Error("Is: unexpected false")
			}
		})
	})
}
