package seccomp

/*
#cgo linux pkg-config: --static libseccomp

#include "libseccomp-helper.h"
#include <sys/personality.h>
*/
import "C"
import (
	"errors"
	"fmt"
	"runtime"
	"runtime/cgo"
	"syscall"
	"unsafe"
)

// ErrInvalidRules is returned for a zero-length rules slice.
var ErrInvalidRules = errors.New("invalid native rules slice")

// LibraryError represents a libseccomp error.
type LibraryError struct {
	// User facing description of the libseccomp function returning the error.
	Prefix string
	// Negated errno value returned by libseccomp.
	Seccomp syscall.Errno
	// Global errno value on return.
	Errno error
}

func (e *LibraryError) Error() string {
	if e.Seccomp == 0 {
		if e.Errno == nil {
			panic("invalid libseccomp error")
		}
		return fmt.Sprintf("%s: %s", e.Prefix, e.Errno)
	}
	if e.Errno == nil {
		return fmt.Sprintf("%s: %s", e.Prefix, e.Seccomp)
	}
	return fmt.Sprintf("%s: %s (%s)", e.Prefix, e.Seccomp, e.Errno)
}

func (e *LibraryError) Is(err error) bool {
	if e == nil {
		return err == nil
	}
	if ef, ok := err.(*LibraryError); ok {
		return *e == *ef
	}
	return (e.Seccomp != 0 && errors.Is(err, e.Seccomp)) ||
		(e.Errno != nil && errors.Is(err, e.Errno))
}

type (
	// ScmpSyscall represents a syscall number passed to libseccomp via [NativeRule.Syscall].
	ScmpSyscall C.int
	// ScmpErrno represents an errno value passed to libseccomp via [NativeRule.Errno].
	ScmpErrno C.int

	// A NativeRule specifies an arch-specific action taken by seccomp under certain conditions.
	NativeRule struct {
		// Syscall is the arch-dependent syscall number to act against.
		Syscall ScmpSyscall
		// Errno is the errno value to return when the condition is satisfied.
		Errno ScmpErrno
		// Arg is the optional struct scmp_arg_cmp passed to libseccomp.
		Arg *ScmpArgCmp
	}

	// syscallRule is equivalent to [NativeRule].
	syscallRule = C.struct_hakurei_syscall_rule
)

// ExportFlag configures filter behaviour that are not implemented as rules.
type ExportFlag = C.hakurei_export_flag

const (
	// AllowMultiarch allows multiarch/emulation.
	AllowMultiarch ExportFlag = C.HAKUREI_EXPORT_MULTIARCH
	// AllowCAN allows AF_CAN.
	AllowCAN ExportFlag = C.HAKUREI_EXPORT_CAN
	// AllowBluetooth allows AF_BLUETOOTH.
	AllowBluetooth ExportFlag = C.HAKUREI_EXPORT_BLUETOOTH
)

var resPrefix = [...]string{
	0: "",
	1: "seccomp_init failed",
	2: "seccomp_arch_add failed",
	3: "seccomp_arch_add failed (multiarch)",
	4: "internal libseccomp failure",
	5: "seccomp_rule_add failed",
	6: "seccomp_export_bpf_mem failed",
	7: "seccomp_load failed",
}

// cbAllocateBuffer is the function signature for the function handle passed to hakurei_export_filter
// which allocates the buffer that the resulting bpf program is copied into, and writes its slice header
// to a value held by the caller.
type cbAllocateBuffer = func(len C.size_t) (buf unsafe.Pointer)

//export hakurei_scmp_allocate
func hakurei_scmp_allocate(f C.uintptr_t, len C.size_t) (buf unsafe.Pointer) {
	return cgo.Handle(f).Value().(cbAllocateBuffer)(len)
}

// makeFilter generates a bpf program from a slice of [NativeRule] and writes the resulting byte slice to p.
// The filter is installed to the current process if p is nil.
func makeFilter(rules []NativeRule, flags ExportFlag, p *[]byte) error {
	if len(rules) == 0 {
		return ErrInvalidRules
	}

	var (
		arch      C.uint32_t = 0
		multiarch C.uint32_t = 0
	)
	switch runtime.GOARCH {
	case "386":
		arch = C.SCMP_ARCH_X86
	case "amd64":
		arch = C.SCMP_ARCH_X86_64
		multiarch = C.SCMP_ARCH_X86
	case "arm":
		arch = C.SCMP_ARCH_ARM
	case "arm64":
		arch = C.SCMP_ARCH_AARCH64
		multiarch = C.SCMP_ARCH_ARM
	}

	var ret C.int

	var scmpPinner runtime.Pinner
	for i := range rules {
		rule := &rules[i]
		scmpPinner.Pin(rule)
		if rule.Arg != nil {
			scmpPinner.Pin(rule.Arg)
		}
	}

	var allocateP cgo.Handle
	if p != nil {
		allocateP = cgo.NewHandle(func(len C.size_t) (buf unsafe.Pointer) {
			// this is so the slice header gets a Go pointer
			*p = make([]byte, len)

			buf = unsafe.Pointer(unsafe.SliceData(*p))
			scmpPinner.Pin(buf)
			return
		})
	}

	res, err := C.hakurei_scmp_make_filter(
		&ret, C.uintptr_t(allocateP),
		arch, multiarch,
		(*syscallRule)(unsafe.Pointer(&rules[0])),
		C.size_t(len(rules)),
		flags,
	)
	scmpPinner.Unpin()
	if p != nil {
		allocateP.Delete()
	}

	if prefix := resPrefix[res]; prefix != "" {
		return &LibraryError{prefix, syscall.Errno(-ret), err}
	}
	return err
}

// Export generates a bpf program from a slice of [NativeRule].
// Errors returned by libseccomp is wrapped in [LibraryError].
func Export(rules []NativeRule, flags ExportFlag) (data []byte, err error) {
	err = makeFilter(rules, flags, &data)
	return
}

// Load generates a bpf program from a slice of [NativeRule] and enforces it on the current process.
// Errors returned by libseccomp is wrapped in [LibraryError].
func Load(rules []NativeRule, flags ExportFlag) error { return makeFilter(rules, flags, nil) }

// ScmpCompare is the equivalent of scmp_compare;
// Comparison operators
type ScmpCompare = C.enum_scmp_compare

const (
	_SCMP_CMP_MIN = C._SCMP_CMP_MIN

	// not equal
	SCMP_CMP_NE = C.SCMP_CMP_NE
	// less than
	SCMP_CMP_LT = C.SCMP_CMP_LT
	// less than or equal
	SCMP_CMP_LE = C.SCMP_CMP_LE
	// equal
	SCMP_CMP_EQ = C.SCMP_CMP_EQ
	// greater than or equal
	SCMP_CMP_GE = C.SCMP_CMP_GE
	// greater than
	SCMP_CMP_GT = C.SCMP_CMP_GT
	// masked equality
	SCMP_CMP_MASKED_EQ = C.SCMP_CMP_MASKED_EQ

	_SCMP_CMP_MAX = C._SCMP_CMP_MAX
)

type (
	// Argument datum.
	scmpDatum = C.scmp_datum_t

	// ScmpDatum is equivalent to scmp_datum_t.
	ScmpDatum uint64

	// Argument / Value comparison definition.
	scmpArgCmp = C.struct_scmp_arg_cmp

	// ScmpArgCmp is equivalent to struct scmp_arg_cmp.
	ScmpArgCmp struct {
		// argument number, starting at 0
		Arg C.uint
		// the comparison op, e.g. SCMP_CMP_*
		Op ScmpCompare

		DatumA, DatumB ScmpDatum
	}
)

const (
	// PersonaLinux is passed in a [ScmpDatum] for filtering calls to syscall.SYS_PERSONALITY.
	PersonaLinux = C.PER_LINUX
	// PersonaLinux32 is passed in a [ScmpDatum] for filtering calls to syscall.SYS_PERSONALITY.
	PersonaLinux32 = C.PER_LINUX32
)

// syscallResolveName resolves a syscall number by name via seccomp_syscall_resolve_name.
// This function is only for testing the lookup tables and included here for convenience.
func syscallResolveName(s string) (trap int, ok bool) {
	v := C.CString(s)
	trap = int(C.seccomp_syscall_resolve_name(v))
	C.free(unsafe.Pointer(v))
	ok = trap != C.__NR_SCMP_ERROR
	return
}
