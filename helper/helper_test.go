package helper_test

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"git.gensokyo.uk/security/fortify/helper"
)

var (
	wantArgs = []string{
		"unix:path=/run/dbus/system_bus_socket",
		"/tmp/fortify.1971/12622d846cc3fe7b4c10359d01f0eb47/system_bus_socket",
		"--filter",
		"--talk=org.bluez",
		"--talk=org.freedesktop.Avahi",
		"--talk=org.freedesktop.UPower",
	}

	wantPayload = strings.Join(wantArgs, "\x00") + "\x00"
	argsWt      = helper.MustNewCheckedArgs(wantArgs)
)

func argF(argsFD, statFD int) []string {
	if argsFD == -1 {
		panic("invalid args fd")
	}

	return argFChecked(argsFD, statFD)
}

func argFChecked(argsFD, statFD int) []string {
	if statFD == -1 {
		return []string{"--args", strconv.Itoa(argsFD)}
	} else {
		return []string{"--args", strconv.Itoa(argsFD), "--fd", strconv.Itoa(statFD)}
	}
}

// this function tests an implementation of the helper.Helper interface
func testHelper(t *testing.T, createHelper func() helper.Helper) {
	helper.InternalReplaceExecCommand(t)

	t.Run("start helper with status channel and wait", func(t *testing.T) {
		h := createHelper()
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

	t.Run("start helper and wait", func(t *testing.T) {
		h := createHelper()
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
