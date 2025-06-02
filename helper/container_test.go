package helper_test

import (
	"context"
	"io"
	"os"
	"os/exec"
	"testing"

	"git.gensokyo.uk/security/fortify/helper"
	"git.gensokyo.uk/security/fortify/internal"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
	"git.gensokyo.uk/security/fortify/sandbox"
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
		if got := helper.New(t.Context(), "fortify", argsWt, false, argF, nil, nil); got == nil {
			t.Errorf("New(%q, %q) got nil",
				argsWt, "fortify")
			return
		}
	})

	t.Run("implementation compliance", func(t *testing.T) {
		testHelper(t, func(ctx context.Context, setOutput func(stdoutP, stderrP *io.Writer), stat bool) helper.Helper {
			return helper.New(ctx, os.Args[0], argsWt, stat, argF, func(container *sandbox.Container) {
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
	sandbox.SetOutput(fmsg.Output{})
	sandbox.Init(fmsg.Prepare, func(bool) { internal.InstallFmsg(false) })
}
