package helper_test

import (
	"context"
	"io"
	"os"
	"os/exec"
	"testing"

	"git.gensokyo.uk/security/hakurei"
	"git.gensokyo.uk/security/hakurei/helper"
	"git.gensokyo.uk/security/hakurei/internal"
	"git.gensokyo.uk/security/hakurei/internal/hlog"
)

func TestContainer(t *testing.T) {
	t.Run("start empty container", func(t *testing.T) {
		h := helper.New(t.Context(), "/nonexistent", argsWt, false, argF, nil, nil)

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
			return helper.New(ctx, os.Args[0], argsWt, stat, argF, func(container *hakurei.Container) {
				setOutput(&container.Stdout, &container.Stderr)
				container.CommandContext = func(ctx context.Context) (cmd *exec.Cmd) {
					return exec.CommandContext(ctx, os.Args[0], "-test.v",
						"-test.run=TestHelperInit", "--", "init")
				}
				container.Bind("/", "/", 0)
				container.Proc("/proc")
				container.Dev("/dev")
			}, nil)
		})
	})
}

func TestHelperInit(t *testing.T) {
	if len(os.Args) != 5 || os.Args[4] != "init" {
		return
	}
	hakurei.SetOutput(hlog.Output{})
	hakurei.Init(hlog.Prepare, func(bool) { internal.InstallOutput(false) })
}
