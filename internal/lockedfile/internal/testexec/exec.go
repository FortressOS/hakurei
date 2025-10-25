package testexec

import (
	"context"
	"os/exec"
	"syscall"
	"testing"
)

// CommandContext is like exec.CommandContext, but:
//   - sends SIGQUIT instead of SIGKILL in its Cancel function
//   - fails the test if the command does not complete before the context is canceled, and
//   - sets a Cleanup function that verifies that the test did not leak a subprocess.
func CommandContext(t testing.TB, ctx context.Context, name string, args ...string) *exec.Cmd {
	t.Helper()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Cancel = func() error {
		if ctx.Err() == context.DeadlineExceeded {
			// The command timed out due to running too close to the test's deadline.
			// There is no way the test did that intentionally — it's too close to the
			// wire! — so mark it as a test failure. That way, if the test expects the
			// command to fail for some other reason, it doesn't have to distinguish
			// between that reason and a timeout.
			t.Errorf("test timed out while running command: %v", cmd)
		} else {
			// The command is being terminated due to ctx being canceled, but
			// apparently not due to an explicit test deadline that we added.
			// Log that information in case it is useful for diagnosing a failure,
			// but don't actually fail the test because of it.
			t.Logf("%v: terminating command: %v", ctx.Err(), cmd)
		}
		return cmd.Process.Signal(syscall.SIGQUIT)
	}

	t.Cleanup(func() {
		if cmd.Process != nil && cmd.ProcessState == nil {
			t.Errorf("command was started, but test did not wait for it to complete: %v", cmd)
		}
	})

	return cmd
}
