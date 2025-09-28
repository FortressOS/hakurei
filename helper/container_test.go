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
	t.Run("start invalid container", func(t *testing.T) {
		h := helper.New(t.Context(), nil, container.MustAbs(container.Nonexistent), "hakurei", argsWt, false, argF, nil, nil)

		wantErr := "container: starting an invalid container"
		if err := h.Start(); err == nil || err.Error() != wantErr {
			t.Errorf("Start: error = %v, wantErr %q",
				err, wantErr)
		}
	})

	t.Run("valid new helper nil check", func(t *testing.T) {
		if got := helper.New(t.Context(), nil, container.MustAbs(container.Nonexistent), "hakurei", argsWt, false, argF, nil, nil); got == nil {
			t.Errorf("New(%q, %q) got nil",
				argsWt, "hakurei")
			return
		}
	})

	t.Run("implementation compliance", func(t *testing.T) {
		testHelper(t, func(ctx context.Context, setOutput func(stdoutP, stderrP *io.Writer), stat bool) helper.Helper {
			return helper.New(ctx, nil, container.MustAbs(os.Args[0]), "helper", argsWt, stat, argF, func(z *container.Container) {
				setOutput(&z.Stdout, &z.Stderr)
				z.
					Bind(container.AbsFHSRoot, container.AbsFHSRoot, 0).
					Proc(container.AbsFHSProc).
					Dev(container.AbsFHSDev, true)
			}, nil)
		})
	})
}
