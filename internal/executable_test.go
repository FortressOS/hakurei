package internal_test

import (
	"os"
	"testing"

	"git.gensokyo.uk/security/fortify/internal"
)

func TestExecutable(t *testing.T) {
	for i := 0; i < 16; i++ {
		if got := internal.MustExecutable(); got != os.Args[0] {
			t.Errorf("MustExecutable: %q, want %q",
				got, os.Args[0])
		}
	}
}
