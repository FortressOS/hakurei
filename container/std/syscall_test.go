package std_test

import (
	"testing"

	"hakurei.app/container/std"
)

func TestSyscallResolveName(t *testing.T) {
	t.Parallel()

	for name, want := range std.Syscalls() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got, ok := std.SyscallResolveName(name); !ok || got != want {
				t.Errorf("SyscallResolveName(%q) = %d, want %d", name, got, want)
			}
		})
	}
}
