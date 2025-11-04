package seccomp

import (
	"reflect"
	"testing"
	"unsafe"

	"hakurei.app/container/std"
)

func TestSyscallResolveName(t *testing.T) {
	t.Parallel()

	for name, want := range std.Syscalls() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// this checks the std implementation against libseccomp.
			if got, ok := syscallResolveName(name); !ok || got != want {
				t.Errorf("syscallResolveName(%q) = %d, want %d", name, got, want)
			}
		})
	}
}

func TestRuleSize(t *testing.T) {
	assertSize[NativeRule, syscallRule](t)
	assertSize[ScmpDatum, scmpDatum](t)
	assertSize[ScmpArgCmp, scmpArgCmp](t)
}

// assertSize asserts that native and equivalent are of the same size.
func assertSize[native, equivalent any](t *testing.T) {
	got := unsafe.Sizeof(*new(native))
	want := unsafe.Sizeof(*new(equivalent))
	if got != want {
		t.Fatalf("%s: %d, want %d", reflect.TypeFor[native]().Name(), got, want)
	}
}
