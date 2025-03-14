package helper_test

import (
	"context"
	"errors"
	"os"
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
			sc, "fortify", false,
			argsWt, argF,
			nil, nil,
		)

		if err := h.Start(false); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("Start: error = %v, wantErr %v",
				err, os.ErrNotExist)
		}
	})

	t.Run("valid new helper nil check", func(t *testing.T) {
		if got := helper.MustNewBwrap(
			context.TODO(),
			sc, "fortify", false,
			argsWt, argF,
			nil, nil,
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
			&bwrap.Config{Hostname: "\x00"}, "fortify", false,
			nil, argF,
			nil, nil,
		)
	})

	t.Run("start without pipes", func(t *testing.T) {
		helper.InternalReplaceExecCommand(t)

		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		h := helper.MustNewBwrap(
			c,
			sc, "crash-test-dummy", false,
			nil, argFChecked,
			nil, nil,
		)

		stdout, stderr := new(strings.Builder), new(strings.Builder)
		h.SetStdout(stdout).SetStderr(stderr)

		if err := h.Start(false); err != nil {
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
		testHelper(t, func(ctx context.Context) helper.Helper {
			return helper.MustNewBwrap(
				ctx,
				sc, "crash-test-dummy", false,
				argsWt, argF, nil, nil,
			)
		})
	})
}
