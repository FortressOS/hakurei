package ldd_test

import (
	"errors"
	"os"
	"os/exec"
	"testing"

	"hakurei.app/container"
	"hakurei.app/ldd"
	"hakurei.app/message"
)

func TestExec(t *testing.T) {
	t.Parallel()

	t.Run("failure", func(t *testing.T) {
		t.Parallel()

		_, err := ldd.Exec(t.Context(), nil, "/proc/nonexistent")

		var exitError *exec.ExitError
		if !errors.As(err, &exitError) {
			t.Fatalf("Exec: error has incorrect concrete type: %#v", err)
		}

		const want = 1
		if got := exitError.ExitCode(); got != want {
			t.Fatalf("Exec: ExitCode = %d, want %d", got, want)
		}
	})

	t.Run("success", func(t *testing.T) {
		msg := message.New(nil)
		msg.GetLogger().SetPrefix("check: ")
		if entries, err := ldd.Exec(t.Context(), nil, container.MustExecutable(msg)); err != nil {
			t.Fatalf("Exec: error = %v", err)
		} else if testing.Verbose() {
			// result cannot be measured here as build information is not known
			t.Logf("Exec: %q", entries)
		}
	})
}

func TestMain(m *testing.M) { container.TryArgv0(nil); os.Exit(m.Run()) }
