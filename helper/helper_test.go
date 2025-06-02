package helper_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
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
	args = make([]string, 0, 6)
	args = append(args, "-test.run=TestHelperStub", "--")
	if argsFd > -1 {
		args = append(args, "--args", strconv.Itoa(argsFd))
	}
	if statFd > -1 {
		args = append(args, "--fd", strconv.Itoa(statFd))
	}
	return
}

// this function tests an implementation of the helper.Helper interface
func testHelper(t *testing.T, createHelper func(ctx context.Context, setOutput func(stdoutP, stderrP *io.Writer), stat bool) helper.Helper) {
	oldWaitDelay := helper.WaitDelay
	helper.WaitDelay = 16 * time.Second
	t.Cleanup(func() { helper.WaitDelay = oldWaitDelay })

	t.Run("start helper with status channel and wait", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		stdout := new(strings.Builder)
		h := createHelper(ctx, func(stdoutP, stderrP *io.Writer) { *stdoutP, *stderrP = stdout, os.Stderr }, true)

		t.Run("wait not yet started helper", func(t *testing.T) {
			defer func() {
				r := recover()
				if r == nil {
					t.Fatalf("Wait did not panic")
				}
			}()
			panic(fmt.Sprintf("unreachable: %v", h.Wait()))
		})

		t.Log("starting helper stub")
		if err := h.Start(); err != nil {
			t.Errorf("Start: error = %v", err)
			cancel()
			return
		}
		t.Log("cancelling context")
		cancel()

		t.Run("start already started helper", func(t *testing.T) {
			wantErr := "helper: already started"
			if err := h.Start(); err != nil && err.Error() != wantErr {
				t.Errorf("Start: error = %v, wantErr %v",
					err, wantErr)
				return
			}
		})

		t.Log("waiting on helper")
		if err := h.Wait(); !errors.Is(err, context.Canceled) {
			t.Errorf("Wait: error = %v",
				err)
		}

		t.Run("wait already finalised helper", func(t *testing.T) {
			wantErr := "exec: Wait was already called"
			if err := h.Wait(); err != nil && err.Error() != wantErr {
				t.Errorf("Wait: error = %v, wantErr %v",
					err, wantErr)
				return
			}
		})

		if got := trimStdout(stdout); got != wantPayload {
			t.Errorf("Start: stdout = %q, want %q",
				got, wantPayload)
		}
	})

	t.Run("start helper and wait", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		stdout := new(strings.Builder)
		h := createHelper(ctx, func(stdoutP, stderrP *io.Writer) { *stdoutP, *stderrP = stdout, os.Stderr }, false)

		if err := h.Start(); err != nil {
			t.Errorf("Start: error = %v",
				err)
			return
		}

		if err := h.Wait(); err != nil {
			t.Errorf("Wait: error = %v stdout = %q",
				err, stdout)
		}

		if got := trimStdout(stdout); got != wantPayload {
			t.Errorf("Start: stdout = %q, want %q",
				got, wantPayload)
		}
	})
}

func trimStdout(stdout fmt.Stringer) string {
	return strings.TrimPrefix(stdout.String(), "=== RUN   TestHelperInit\n")
}
