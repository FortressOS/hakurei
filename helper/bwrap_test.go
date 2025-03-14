package helper_test

import (
	"context"
	"errors"
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
		Chdir:         "/nonexistent",
		Clearenv:      true,
		NewSession:    true,
		DieWithParent: true,
		AsInit:        true,
	}

	t.Run("nonexistent bwrap name", func(t *testing.T) {
		bubblewrapName := helper.BubblewrapName
		helper.BubblewrapName = "/nonexistent"
		t.Cleanup(func() {
			helper.BubblewrapName = bubblewrapName
		})

		h := helper.MustNewBwrap(
			context.Background(),
			"fortify",
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
			"fortify",
			argsWt, false,
			argF, nil,
			nil,
			sc, nil,
		); got == nil {
			t.Errorf("MustNewBwrap(%#v, %#v, %#v) got nil",
				sc, argsWt, "fortify")
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
			"fortify",
			argsWt, false,
			argF, nil,
			nil,
			&bwrap.Config{Hostname: "\x00"}, nil,
		)
	})

	t.Run("start without pipes", func(t *testing.T) {
		helper.InternalReplaceExecCommand(t)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		stdout, stderr := new(strings.Builder), new(strings.Builder)
		h := helper.MustNewBwrap(
			ctx, "crash-test-dummy",
			nil, false,
			argFChecked, func(cmd *exec.Cmd) { cmd.Stdout, cmd.Stderr = stdout, stderr },
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
		testHelper(t, func(ctx context.Context, cmdF func(cmd *exec.Cmd), stat bool) helper.Helper {
			return helper.MustNewBwrap(
				ctx, "crash-test-dummy",
				argsWt, stat,
				argF, cmdF,
				nil,
				sc, nil,
			)
		})
	})
}
