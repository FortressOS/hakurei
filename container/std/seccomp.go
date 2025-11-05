package std

import (
	"encoding/json"
	"strconv"
)

type (
	// ScmpUint is equivalent to C.uint.
	ScmpUint uint32
	// ScmpInt is equivalent to C.int.
	ScmpInt int32

	// ScmpSyscall represents a syscall number passed to libseccomp via [NativeRule.Syscall].
	ScmpSyscall ScmpInt
	// ScmpErrno represents an errno value passed to libseccomp via [NativeRule.Errno].
	ScmpErrno ScmpInt

	// ScmpCompare is equivalent to enum scmp_compare;
	ScmpCompare ScmpUint
	// ScmpDatum is equivalent to scmp_datum_t.
	ScmpDatum uint64

	// ScmpArgCmp is equivalent to struct scmp_arg_cmp.
	ScmpArgCmp struct {
		// argument number, starting at 0
		Arg ScmpUint `json:"arg"`
		// the comparison op, e.g. SCMP_CMP_*
		Op ScmpCompare `json:"op"`

		DatumA ScmpDatum `json:"a,omitempty"`
		DatumB ScmpDatum `json:"b,omitempty"`
	}

	// A NativeRule specifies an arch-specific action taken by seccomp under certain conditions.
	NativeRule struct {
		// Syscall is the arch-dependent syscall number to act against.
		Syscall ScmpSyscall `json:"syscall"`
		// Errno is the errno value to return when the condition is satisfied.
		Errno ScmpErrno `json:"errno"`
		// Arg is the optional struct scmp_arg_cmp passed to libseccomp.
		Arg *ScmpArgCmp `json:"arg,omitempty"`
	}
)

// MarshalJSON resolves the name of [ScmpSyscall] and encodes it as a [json] string.
// If such a name does not exist, the syscall number is encoded instead.
func (num *ScmpSyscall) MarshalJSON() ([]byte, error) {
	n := int(*num)
	for name, cur := range Syscalls() {
		if cur == n {
			return json.Marshal(name)
		}
	}
	return json.Marshal(n)
}

// SyscallNameError is returned when trying to unmarshal an invalid syscall name into [ScmpSyscall].
type SyscallNameError string

func (e SyscallNameError) Error() string { return "invalid syscall name " + strconv.Quote(string(e)) }

// UnmarshalJSON looks up the syscall number corresponding to name encoded in data
// by calling [SyscallResolveName].
func (num *ScmpSyscall) UnmarshalJSON(data []byte) error {
	var name string
	if err := json.Unmarshal(data, &name); err != nil {
		return err
	}
	if n, ok := SyscallResolveName(name); !ok {
		return SyscallNameError(name)
	} else {
		*num = ScmpSyscall(n)
		return nil
	}
}
