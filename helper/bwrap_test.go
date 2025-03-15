package helper_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"git.gensokyo.uk/security/fortify/helper"
	"git.gensokyo.uk/security/fortify/helper/bwrap"
)

func TestBwrap(t *testing.T) {
	sc := &bwrap.Config{
		Net:           true,
		Hostname:      "localhost",
		Chdir:         "/proc/nonexistent",
		Clearenv:      true,
		NewSession:    true,
		DieWithParent: true,
		AsInit:        true,
	}

	t.Run("nonexistent bwrap name", func(t *testing.T) {
		bubblewrapName := helper.BubblewrapName
		helper.BubblewrapName = "/proc/nonexistent"
		t.Cleanup(func() { helper.BubblewrapName = bubblewrapName })

		h := helper.MustNewBwrap(
			context.Background(),
			"false",
			argsWt, false,
			argF, nil,
			nil,
			sc, nil,
		)

		if err := h.Start(); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("Start: error = %v, wantErr %v",
				err, os.ErrNotExist)
		}
	})

	t.Run("valid new helper nil check", func(t *testing.T) {
		if got := helper.MustNewBwrap(
			context.TODO(),
			"false",
			argsWt, false,
			argF, nil,
			nil,
			sc, nil,
		); got == nil {
			t.Errorf("MustNewBwrap(%#v, %#v, %#v) got nil",
				sc, argsWt, "false")
			return
		}
	})

	t.Run("invalid bwrap config new helper panic", func(t *testing.T) {
		defer func() {
			wantPanic := "argument contains null character"
			if r := recover(); r != wantPanic {
				t.Errorf("MustNewBwrap: panic = %q, want %q",
					r, wantPanic)
			}
		}()

		helper.MustNewBwrap(
			context.TODO(),
			"false",
			argsWt, false,
			argF, nil,
			nil,
			&bwrap.Config{Hostname: "\x00"}, nil,
		)
	})

	t.Run("start without pipes", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		stdout, stderr := new(strings.Builder), new(strings.Builder)
		h := helper.MustNewBwrap(
			ctx, os.Args[0],
			nil, false,
			argFChecked, func(cmd *exec.Cmd) { cmd.Stdout, cmd.Stderr = stdout, stderr; hijackBwrap(cmd) },
			nil,
			sc, nil,
		)

		if err := h.Start(); err != nil {
			t.Errorf("Start: error = %v",
				err)
			return
		}

		if err := h.Wait(); err != nil {
			t.Errorf("Wait() err = %v stderr = %s",
				err, stderr)
		}
	})

	t.Run("implementation compliance", func(t *testing.T) {
		testHelper(t, func(ctx context.Context, setOutput func(stdoutP, stderrP *io.Writer), stat bool) helper.Helper {
			return helper.MustNewBwrap(
				ctx, os.Args[0],
				argsWt, stat,
				argF, func(cmd *exec.Cmd) { setOutput(&cmd.Stdout, &cmd.Stderr); hijackBwrap(cmd) },
				nil,
				sc, nil,
			)
		})
	})
}

func hijackBwrap(cmd *exec.Cmd) {
	if cmd.Args[0] != "bwrap" {
		panic(fmt.Sprintf("unexpected argv0 %q", cmd.Args[0]))
	}
	cmd.Err = nil
	cmd.Path = os.Args[0]
	cmd.Args = append([]string{os.Args[0], "-test.run=TestHelperStub", "--"}, cmd.Args...)
}
