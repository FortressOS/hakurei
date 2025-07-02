package helper_test

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"testing"

	"hakurei.app/helper"
)

func TestCmd(t *testing.T) {
	t.Run("start non-existent helper path", func(t *testing.T) {
		h := helper.NewDirect(t.Context(), "/proc/nonexistent", argsWt, false, argF, nil, nil)

		if err := h.Start(); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("Start: error = %v, wantErr %v",
				err, os.ErrNotExist)
		}
	})

	t.Run("valid new helper nil check", func(t *testing.T) {
		if got := helper.NewDirect(t.Context(), "hakurei", argsWt, false, argF, nil, nil); got == nil {
			t.Errorf("NewDirect(%q, %q) got nil",
				argsWt, "hakurei")
			return
		}
	})

	t.Run("implementation compliance", func(t *testing.T) {
		testHelper(t, func(ctx context.Context, setOutput func(stdoutP, stderrP *io.Writer), stat bool) helper.Helper {
			return helper.NewDirect(ctx, os.Args[0], argsWt, stat, argF, func(cmd *exec.Cmd) {
				setOutput(&cmd.Stdout, &cmd.Stderr)
			}, nil)
		})
	})
}
