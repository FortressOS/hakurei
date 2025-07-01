package seccomp

/*
#cgo linux pkg-config: --static libseccomp

#include <libseccomp-helper.h>
*/
import "C"
import (
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
)

var (
	ErrInvalidRules = errors.New("invalid native rules slice")
)

// LibraryError represents a libseccomp error.
type LibraryError struct {
	Prefix  string
	Seccomp syscall.Errno
	Errno   error
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
	ScmpSyscall = C.int
	ScmpErrno   = C.int
)

// A NativeRule specifies an arch-specific action taken by seccomp under certain conditions.
type NativeRule struct {
	// Syscall is the arch-dependent syscall number to act against.
	Syscall ScmpSyscall
	// Errno is the errno value to return when the condition is satisfied.
	Errno ScmpErrno
	// Arg is the optional struct scmp_arg_cmp passed to libseccomp.
	Arg *ScmpArgCmp
}

type PrepareFlag = C.hakurei_prepare_flag

const (
	// AllowMultiarch allows multiarch/emulation.
	AllowMultiarch PrepareFlag = C.HAKUREI_PREPARE_MULTIARCH
	// AllowCAN allows AF_CAN.
	AllowCAN PrepareFlag = C.HAKUREI_PREPARE_CAN
	// AllowBluetooth allows AF_BLUETOOTH.
	AllowBluetooth PrepareFlag = C.HAKUREI_PREPARE_BLUETOOTH
)

var resPrefix = [...]string{
	0: "",
	1: "seccomp_init failed",
	2: "seccomp_arch_add failed",
	3: "seccomp_arch_add failed (multiarch)",
	4: "internal libseccomp failure",
	5: "seccomp_rule_add failed",
	6: "seccomp_export_bpf failed",
	7: "seccomp_load failed",
}

// Prepare streams filter contents to fd, or installs it to the current process if fd < 0.
func Prepare(fd int, rules []NativeRule, flags PrepareFlag) error {
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

	rulesPinner := new(runtime.Pinner)
	for i := range rules {
		rule := &rules[i]
		rulesPinner.Pin(rule)
		if rule.Arg != nil {
			rulesPinner.Pin(rule.Arg)
		}
	}
	res, err := C.hakurei_prepare_filter(
		&ret, C.int(fd),
		arch, multiarch,
		(*C.struct_hakurei_syscall_rule)(unsafe.Pointer(&rules[0])),
		C.size_t(len(rules)),
		flags,
	)
	rulesPinner.Unpin()

	if prefix := resPrefix[res]; prefix != "" {
		return &LibraryError{
			prefix,
			-syscall.Errno(ret),
			err,
		}
	}
	return err
}

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

// ScmpDatum is the equivalent of scmp_datum_t;
// Argument datum
type ScmpDatum uint64

// ScmpArgCmp is the equivalent of struct scmp_arg_cmp;
// Argument / Value comparison definition
type ScmpArgCmp struct {
	// argument number, starting at 0
	arg C.uint
	// the comparison op, e.g. SCMP_CMP_*
	op ScmpCompare

	datum_a, datum_b ScmpDatum
}

// only used for testing
func syscallResolveName(s string) (trap int) {
	v := C.CString(s)
	trap = int(C.seccomp_syscall_resolve_name(v))
	C.free(unsafe.Pointer(v))

	return
}
