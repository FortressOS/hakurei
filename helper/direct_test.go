package helper_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"git.gensokyo.uk/security/fortify/helper"
)

func TestDirect(t *testing.T) {
	t.Run("start non-existent helper path", func(t *testing.T) {
		h := helper.New(context.Background(), argsWt, "/nonexistent", argF)

		if err := h.Start(false); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("Start: error = %v, wantErr %v",
				err, os.ErrNotExist)
		}
	})

	t.Run("valid new helper nil check", func(t *testing.T) {
		if got := helper.New(context.TODO(), argsWt, "fortify", argF); got == nil {
			t.Errorf("New(%q, %q) got nil",
				argsWt, "fortify")
			return
		}
	})

	t.Run("implementation compliance", func(t *testing.T) {
		testHelper(t, func(ctx context.Context) helper.Helper { return helper.New(ctx, argsWt, "crash-test-dummy", argF) })
	})
}
