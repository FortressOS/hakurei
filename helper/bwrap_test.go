package helper_test

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"git.gensokyo.uk/security/fortify/helper"
	"git.gensokyo.uk/security/fortify/helper/bwrap"
)

func TestBwrap(t *testing.T) {
	sc := &bwrap.Config{
		Unshare:       nil,
		Net:           true,
		UserNS:        false,
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
			sc, "fortify",
			argsWt, argF,
			nil, nil,
		)

		if err := h.Start(); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("Start() error = %v, wantErr %v",
				err, os.ErrNotExist)
		}
	})

	t.Run("valid new helper nil check", func(t *testing.T) {
		if got := helper.MustNewBwrap(
			sc, "fortify",
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
			&bwrap.Config{Hostname: "\x00"}, "fortify",
			nil, argF,
			nil, nil,
		)
	})

	t.Run("start notify without pipes panic", func(t *testing.T) {
		defer func() {
			wantPanic := "attempted to start with status monitoring on a bwrap child initialised without pipes"
			if r := recover(); r != wantPanic {
				t.Errorf("StartNotify: panic = %q, want %q",
					r, wantPanic)
			}
		}()

		panic(fmt.Sprintf("unreachable: %v",
			helper.MustNewBwrap(
				sc, "fortify",
				nil, argF,
				nil, nil,
			).StartNotify(make(chan error))))
	})

	t.Run("start without pipes", func(t *testing.T) {
		helper.InternalReplaceExecCommand(t)

		h := helper.MustNewBwrap(
			sc, "crash-test-dummy",
			nil, argFChecked,
			nil, nil,
		)
		cmd := h.Unwrap()

		stdout, stderr := new(strings.Builder), new(strings.Builder)
		cmd.Stdout, cmd.Stderr = stdout, stderr

		t.Run("close without pipes panic", func(t *testing.T) {
			defer func() {
				wantPanic := "attempted to close bwrap child initialised without pipes"
				if r := recover(); r != wantPanic {
					t.Errorf("Close: panic = %q, want %q",
						r, wantPanic)
				}
			}()

			panic(fmt.Sprintf("unreachable: %v",
				h.Close()))
		})

		if err := h.Start(); err != nil {
			t.Errorf("Start() error = %v",
				err)
			return
		}

		if err := h.Wait(); err != nil {
			t.Errorf("Wait() err = %v stderr = %s",
				err, stderr)
		}
	})

	t.Run("implementation compliance", func(t *testing.T) {
		testHelper(t, func() helper.Helper { return helper.MustNewBwrap(sc, "crash-test-dummy", argsWt, argF, nil, nil) })
	})
}
