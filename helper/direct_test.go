package helper_test

import (
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"git.ophivana.moe/cat/fortify/helper"
)

var (
	want = []string{
		"unix:path=/run/dbus/system_bus_socket",
		"/tmp/fortify.1971/12622d846cc3fe7b4c10359d01f0eb47/system_bus_socket",
		"--filter",
		"--talk=org.bluez",
		"--talk=org.freedesktop.Avahi",
		"--talk=org.freedesktop.UPower",
	}

	wantPayload = strings.Join(want, "\x00") + "\x00"
	argsWt      = helper.MustNewCheckedArgs(want)
)

func argF(argsFD int, _ int) []string {
	return []string{"--args", strconv.Itoa(argsFD)}
}

func argFStatus(argsFD int, statFD int) []string {
	return []string{"--args", strconv.Itoa(argsFD), "--fd", strconv.Itoa(statFD)}
}

func TestHelper_StartNotify_Close_Wait(t *testing.T) {
	helper.InternalReplaceExecCommand(t)

	t.Run("start non-existent helper path", func(t *testing.T) {
		h := helper.New(argsWt, "/nonexistent", argF)

		if err := h.Start(); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("Start() error = %v, wantErr %v",
				err, os.ErrNotExist)
		}
	})

	t.Run("start helper with status channel", func(t *testing.T) {
		h := helper.New(argsWt, "crash-test-dummy", argFStatus)
		ready := make(chan error, 1)
		cmd := h.Unwrap()

		stdout, stderr := new(strings.Builder), new(strings.Builder)
		cmd.Stdout, cmd.Stderr = stdout, stderr

		t.Run("wait not yet started helper", func(t *testing.T) {
			wantErr := "exec: not started"
			if err := h.Wait(); err != nil && err.Error() != wantErr {
				t.Errorf("Wait(%v) error = %v, wantErr %v",
					ready,
					err, wantErr)
				return
			}
		})

		t.Log("starting helper stub")
		if err := h.StartNotify(ready); err != nil {
			t.Errorf("StartNotify(%v) error = %v",
				ready,
				err)
			return
		}

		t.Run("start already started helper", func(t *testing.T) {
			wantErr := "exec: already started"
			if err := h.StartNotify(ready); err != nil && err.Error() != wantErr {
				t.Errorf("StartNotify(%v) error = %v, wantErr %v",
					ready,
					err, wantErr)
				return
			}
		})

		t.Log("waiting on status channel with timeout")
		select {
		case <-time.NewTimer(5 * time.Second).C:
			t.Errorf("never got a ready response")
			t.Errorf("stdout:\n%s", stdout.String())
			t.Errorf("stderr:\n%s", stderr.String())
			if err := cmd.Process.Kill(); err != nil {
				panic(err.Error())
			}
			return
		case err := <-ready:
			if err != nil {
				t.Errorf("StartNotify(%v) latent error = %v",
					ready,
					err)
			}
		}

		t.Log("closing status pipe")
		if err := h.Close(); err != nil {
			t.Errorf("Close() error = %v",
				err)
		}

		t.Log("waiting on helper")
		if err := h.Wait(); err != nil {
			t.Errorf("Wait() err = %v stderr = %s",
				err, stderr)
		}

		t.Run("wait already finalised helper", func(t *testing.T) {
			wantErr := "exec: Wait was already called"
			if err := h.Wait(); err != nil && err.Error() != wantErr {
				t.Errorf("Wait(%v) error = %v, wantErr %v",
					ready,
					err, wantErr)
				return
			}
		})

		if got := stdout.String(); !strings.HasPrefix(got, wantPayload) {
			t.Errorf("StartNotify(%v) stdout = %v, want %v",
				ready,
				got, wantPayload)
		}
	})
}
func TestHelper_Start_Close_Wait(t *testing.T) {
	helper.InternalReplaceExecCommand(t)

	var wt io.WriterTo
	if a, err := helper.NewCheckedArgs(want); err != nil {
		t.Errorf("NewCheckedArgs(%q) error = %v",
			want,
			err)
		return
	} else {
		wt = a
	}

	t.Run("start helper", func(t *testing.T) {
		h := helper.New(wt, "crash-test-dummy", argF)
		cmd := h.Unwrap()

		stdout, stderr := new(strings.Builder), new(strings.Builder)
		cmd.Stdout, cmd.Stderr = stdout, stderr

		if err := h.Start(); err != nil {
			t.Errorf("Start() error = %v",
				err)
			return
		}

		t.Run("close helper without status pipe", func(t *testing.T) {
			defer func() {
				wantPanic := "attempted to close helper with no status pipe"
				if r := recover(); r != wantPanic {
					t.Errorf("Close() panic = %v, wantPanic %v",
						r, wantPanic)
				}
			}()
			if err := h.Close(); err != nil {
				t.Errorf("Close() error = %v",
					err)
				return
			}
		})

		if err := h.Wait(); err != nil {
			t.Errorf("Wait() err = %v stderr = %s",
				err, stderr)
		}

		if got := stdout.String(); !strings.HasPrefix(got, wantPayload) {
			t.Errorf("Start() stdout = %v, want %v",
				got, wantPayload)
		}
	})
}

func TestNew(t *testing.T) {
	t.Run("valid new helper nil check", func(t *testing.T) {
		swt, _ := helper.NewCheckedArgs(make([]string, 1))
		if got := helper.New(swt, "fortify", argF); got == nil {
			t.Errorf("New(%q, %q) got nil",
				swt, "fortify")
			return
		}
	})

	t.Run("invalid new helper panic", func(t *testing.T) {
		defer func() {
			want := "attempted to create helper with invalid argument writer"
			if r := recover(); r != want {
				t.Errorf("New: panic = %q, want %q",
					r, want)
			}
		}()

		helper.New(nil, "fortify", argF)
	})
}
