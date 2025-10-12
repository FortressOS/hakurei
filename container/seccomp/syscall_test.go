package seccomp

import (
	"testing"
)

func TestSyscallResolveName(t *testing.T) {
	t.Parallel()

	for name, want := range Syscalls() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := syscallResolveName(name); got != want {
				t.Errorf("syscallResolveName(%q) = %d, want %d", name, got, want)
			}
			if got, ok := SyscallResolveName(name); !ok || got != want {
				t.Errorf("SyscallResolveName(%q) = %d, want %d", name, got, want)
			}
		})
	}
}
