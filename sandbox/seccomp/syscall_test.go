package seccomp

import (
	"testing"
)

func TestSyscallResolveName(t *testing.T) {
	for name, want := range syscallNum {
		t.Run(name, func(t *testing.T) {
			if got := syscallResolveName(name); got != want {
				t.Errorf("syscallResolveName(%q) = %d, want %d",
					name, got, want)
			}
		})
	}
}
