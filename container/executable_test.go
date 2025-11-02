package container_test

import (
	"os"
	"testing"

	"hakurei.app/container"
	"hakurei.app/message"
)

func TestExecutable(t *testing.T) {
	t.Parallel()
	for i := 0; i < 16; i++ {
		if got := container.MustExecutable(message.New(nil)); got != os.Args[0] {
			t.Errorf("MustExecutable: %q, want %q", got, os.Args[0])
		}
	}
}
