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

func TestRuleType(t *testing.T) {
	assertKind[std.ScmpUint, scmpUint](t)
	assertKind[std.ScmpInt, scmpInt](t)

	assertSize[std.NativeRule, syscallRule](t)
	assertKind[std.ScmpDatum, scmpDatum](t)
	assertKind[std.ScmpCompare, scmpCompare](t)
	assertSize[std.ScmpArgCmp, scmpArgCmp](t)
}

// assertSize asserts that native and equivalent are of the same size.
func assertSize[native, equivalent any](t *testing.T) {
	t.Helper()

	got, want := unsafe.Sizeof(*new(native)), unsafe.Sizeof(*new(equivalent))
	if got != want {
		t.Fatalf("%s: %d, want %d", reflect.TypeFor[native]().Name(), got, want)
	}
}

// assertKind asserts that native and equivalent are of the same kind.
func assertKind[native, equivalent any](t *testing.T) {
	t.Helper()

	assertSize[native, equivalent](t)
	nativeType, equivalentType := reflect.TypeFor[native](), reflect.TypeFor[equivalent]()
	got, want := nativeType.Kind(), equivalentType.Kind()

	if got == reflect.Invalid || want == reflect.Invalid {
		t.Fatalf("%s: invalid call to assertKind", nativeType.Name())
	}
	if got == reflect.Struct {
		t.Fatalf("%s: struct is unsupported by assertKind", nativeType.Name())
	}
	if got != want {
		t.Fatalf("%s: %s, want %s", nativeType.Name(), nativeType.Kind(), equivalentType.Kind())
	}
}
