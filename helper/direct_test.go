package helper_test

import (
	"errors"
	"os"
	"testing"

	"git.ophivana.moe/cat/fortify/helper"
)

func TestDirect(t *testing.T) {
	t.Run("start non-existent helper path", func(t *testing.T) {
		h := helper.New(argsWt, "/nonexistent", argF)

		if err := h.Start(); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("Start() error = %v, wantErr %v",
				err, os.ErrNotExist)
		}
	})

	t.Run("valid new helper nil check", func(t *testing.T) {
		if got := helper.New(argsWt, "fortify", argF); got == nil {
			t.Errorf("New(%q, %q) got nil",
				argsWt, "fortify")
			return
		}
	})

	t.Run("invalid new helper panic", func(t *testing.T) {
		defer func() {
			wantPanic := "attempted to create helper with invalid argument writer"
			if r := recover(); r != wantPanic {
				t.Errorf("New: panic = %q, want %q",
					r, wantPanic)
			}
		}()

		helper.New(nil, "fortify", argF)
	})

	t.Run("implementation compliance", func(t *testing.T) {
		testHelper(t, func() helper.Helper { return helper.New(argsWt, "crash-test-dummy", argF) })
	})
}
