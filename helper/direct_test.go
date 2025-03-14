package helper_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"

	"git.gensokyo.uk/security/fortify/helper"
)

func TestDirect(t *testing.T) {
	t.Run("start non-existent helper path", func(t *testing.T) {
		h := helper.NewDirect(context.Background(), "/nonexistent", argsWt, false, argF, nil)

		if err := h.Start(); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("Start: error = %v, wantErr %v",
				err, os.ErrNotExist)
		}
	})

	t.Run("valid new helper nil check", func(t *testing.T) {
		if got := helper.NewDirect(context.TODO(), "fortify", argsWt, false, argF, nil); got == nil {
			t.Errorf("New(%q, %q) got nil",
				argsWt, "fortify")
			return
		}
	})

	t.Run("implementation compliance", func(t *testing.T) {
		testHelper(t, func(ctx context.Context, cmdF func(cmd *exec.Cmd), stat bool) helper.Helper {
			return helper.NewDirect(ctx, "crash-test-dummy", argsWt, stat, argF, cmdF)
		})
	})
}
