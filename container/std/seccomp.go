package std

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
		Arg ScmpUint
		// the comparison op, e.g. SCMP_CMP_*
		Op ScmpCompare

		DatumA, DatumB ScmpDatum
	}

	// A NativeRule specifies an arch-specific action taken by seccomp under certain conditions.
	NativeRule struct {
		// Syscall is the arch-dependent syscall number to act against.
		Syscall ScmpSyscall
		// Errno is the errno value to return when the condition is satisfied.
		Errno ScmpErrno
		// Arg is the optional struct scmp_arg_cmp passed to libseccomp.
		Arg *ScmpArgCmp
	}
)
