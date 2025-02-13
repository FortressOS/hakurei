package helper_test

import (
	"context"
	"errors"
	"fmt"
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

func argF(argsFd, statFd int) []string {
	if argsFd == -1 {
		panic("invalid args fd")
	}

	return argFChecked(argsFd, statFd)
}

func argFChecked(argsFd, statFd int) (args []string) {
	args = make([]string, 0, 4)
	if argsFd > -1 {
		args = append(args, "--args", strconv.Itoa(argsFd))
	}
	if statFd > -1 {
		args = append(args, "--fd", strconv.Itoa(statFd))
	}
	return
}

// this function tests an implementation of the helper.Helper interface
func testHelper(t *testing.T, createHelper func() helper.Helper) {
	helper.InternalReplaceExecCommand(t)

	t.Run("start helper with status channel and wait", func(t *testing.T) {
		h := createHelper()

		stdout, stderr := new(strings.Builder), new(strings.Builder)
		h.Stdout(stdout).Stderr(stderr)

		t.Run("wait not yet started helper", func(t *testing.T) {
			defer func() {
				r := recover()
				if r == nil {
					t.Fatalf("Wait did not panic")
				}
			}()
			panic(fmt.Sprintf("unreachable: %v", h.Wait()))
		})

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		t.Log("starting helper stub")
		if err := h.Start(ctx, true); err != nil {
			t.Errorf("Start: error = %v", err)
			cancel()
			return
		}
		t.Log("cancelling context")
		cancel()

		t.Run("start already started helper", func(t *testing.T) {
			wantErr := "exec: already started"
			if err := h.Start(ctx, true); err != nil && err.Error() != wantErr {
				t.Errorf("Start: error = %v, wantErr %v",
					err, wantErr)
				return
			}
		})

		t.Log("waiting on helper")
		if err := h.Wait(); !errors.Is(err, context.Canceled) {
			t.Errorf("Wait() err = %v stderr = %s",
				err, stderr)
		}

		t.Run("wait already finalised helper", func(t *testing.T) {
			wantErr := "exec: Wait was already called"
			if err := h.Wait(); err != nil && err.Error() != wantErr {
				t.Errorf("Wait: error = %v, wantErr %v",
					err, wantErr)
				return
			}
		})

		if got := stdout.String(); !strings.HasPrefix(got, wantPayload) {
			t.Errorf("Start: stdout = %v, want %v",
				got, wantPayload)
		}
	})

	t.Run("start helper and wait", func(t *testing.T) {
		h := createHelper()

		stdout, stderr := new(strings.Builder), new(strings.Builder)
		h.Stdout(stdout).Stderr(stderr)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := h.Start(ctx, false); err != nil {
			t.Errorf("Start() error = %v",
				err)
			return
		}

		if err := h.Wait(); err != nil {
			t.Errorf("Wait() err = %v stdout = %s stderr = %s",
				err, stdout, stderr)
		}

		if got := stdout.String(); !strings.HasPrefix(got, wantPayload) {
			t.Errorf("Start() stdout = %v, want %v",
				got, wantPayload)
		}
	})
}
