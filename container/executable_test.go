package container_test

import (
	"os"
	"testing"

	"hakurei.app/container"
)

func TestExecutable(t *testing.T) {
	for i := 0; i < 16; i++ {
		if got := container.MustExecutable(); got != os.Args[0] {
			t.Errorf("MustExecutable: %q, want %q",
				got, os.Args[0])
		}
	}
}
