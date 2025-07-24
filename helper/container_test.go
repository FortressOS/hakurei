package helper_test

import (
	"context"
	"io"
	"os"
	"testing"

	"hakurei.app/container"
	"hakurei.app/helper"
)

func TestContainer(t *testing.T) {
	t.Run("start empty container", func(t *testing.T) {
		h := helper.New(t.Context(), container.Nonexistent, argsWt, false, argF, nil, nil)

		wantErr := "sandbox: starting an empty container"
		if err := h.Start(); err == nil || err.Error() != wantErr {
			t.Errorf("Start: error = %v, wantErr %q",
				err, wantErr)
		}
	})

	t.Run("valid new helper nil check", func(t *testing.T) {
		if got := helper.New(t.Context(), "hakurei", argsWt, false, argF, nil, nil); got == nil {
			t.Errorf("New(%q, %q) got nil",
				argsWt, "hakurei")
			return
		}
	})

	t.Run("implementation compliance", func(t *testing.T) {
		testHelper(t, func(ctx context.Context, setOutput func(stdoutP, stderrP *io.Writer), stat bool) helper.Helper {
			return helper.New(ctx, os.Args[0], argsWt, stat, argF, func(z *container.Container) {
				setOutput(&z.Stdout, &z.Stderr)
				z.Bind("/", "/", 0).Proc("/proc").Dev("/dev")
			}, nil)
		})
	})
}
