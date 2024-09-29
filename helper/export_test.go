package helper

import (
	"os"
	"os/exec"
	"testing"
)

// replace execCommand to have the resulting *exec.Cmd launch TestHelperChildStub
func ReplaceExecCommand(t *testing.T) {
	t.Cleanup(func() {
		execCommand = exec.Command
	})

	execCommand = func(name string, arg ...string) *exec.Cmd {
		return exec.Command(os.Args[0], append([]string{"-test.run=TestHelperChildStub", "--", name}, arg...)...)
	}
}
