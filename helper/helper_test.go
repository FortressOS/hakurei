package helper_test

import (
	"io"
	"strings"
	"sync"
	"testing"

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

	wantPayload string
	argsWt      io.WriterTo
	argsOnce    sync.Once
)

func prepareArgs() {
	wantPayload = strings.Join(want, "\x00") + "\x00"

	if a, err := helper.NewCheckedArgs(want); err != nil {
		panic(err.Error())
	} else {
		argsWt = a
	}
}

func TestHelper_StartNotify_Close_Wait(t *testing.T) {
	helper.InternalReplaceExecCommand(t)
	argsOnce.Do(prepareArgs)

	t.Run("start helper with status channel", func(t *testing.T) {
		h := helper.New(argsWt, "crash-test-dummy", "--args=3", "--fd=4")
		ready := make(chan error, 1)

		stdout, stderr := new(strings.Builder), new(strings.Builder)
		h.Stdout, h.Stderr = stdout, stderr

		t.Run("wait not yet started helper", func(t *testing.T) {
			wantErr := "exec: not started"
			if err := h.Wait(); err != nil && err.Error() != wantErr {
				t.Errorf("Wait(%v) error = %v, wantErr %v",
					ready,
					err, wantErr)
				return
			}
		})

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

		if err := <-ready; err != nil {
			t.Errorf("StartNotify(%v) latent error = %v",
				ready,
				err)
		}

		if err := h.Close(); err != nil {
			t.Errorf("Close() error = %v",
				err)
		}

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
	argsOnce.Do(prepareArgs)

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
		h := helper.New(wt, "crash-test-dummy", "--args=3")

		stdout, stderr := new(strings.Builder), new(strings.Builder)
		h.Stdout, h.Stderr = stdout, stderr

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
		if got := helper.New(swt, "fortify"); got == nil {
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

		helper.New(nil, "fortify")
	})
}
