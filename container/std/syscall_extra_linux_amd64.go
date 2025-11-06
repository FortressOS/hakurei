package std

var syscallNumExtra = map[string]ScmpSyscall{
	"umount":          SNR_UMOUNT,
	"subpage_prot":    SNR_SUBPAGE_PROT,
	"switch_endian":   SNR_SWITCH_ENDIAN,
	"vm86":            SNR_VM86,
	"vm86old":         SNR_VM86OLD,
	"clock_adjtime64": SNR_CLOCK_ADJTIME64,
	"clock_settime64": SNR_CLOCK_SETTIME64,
	"chown32":         SNR_CHOWN32,
	"fchown32":        SNR_FCHOWN32,
	"lchown32":        SNR_LCHOWN32,
	"setgid32":        SNR_SETGID32,
	"setgroups32":     SNR_SETGROUPS32,
	"setregid32":      SNR_SETREGID32,
	"setresgid32":     SNR_SETRESGID32,
	"setresuid32":     SNR_SETRESUID32,
	"setreuid32":      SNR_SETREUID32,
	"setuid32":        SNR_SETUID32,
}

const (
	SNR_UMOUNT          ScmpSyscall = __PNR_umount
	SNR_SUBPAGE_PROT    ScmpSyscall = __PNR_subpage_prot
	SNR_SWITCH_ENDIAN   ScmpSyscall = __PNR_switch_endian
	SNR_VM86            ScmpSyscall = __PNR_vm86
	SNR_VM86OLD         ScmpSyscall = __PNR_vm86old
	SNR_CLOCK_ADJTIME64 ScmpSyscall = __PNR_clock_adjtime64
	SNR_CLOCK_SETTIME64 ScmpSyscall = __PNR_clock_settime64
	SNR_CHOWN32         ScmpSyscall = __PNR_chown32
	SNR_FCHOWN32        ScmpSyscall = __PNR_fchown32
	SNR_LCHOWN32        ScmpSyscall = __PNR_lchown32
	SNR_SETGID32        ScmpSyscall = __PNR_setgid32
	SNR_SETGROUPS32     ScmpSyscall = __PNR_setgroups32
	SNR_SETREGID32      ScmpSyscall = __PNR_setregid32
	SNR_SETRESGID32     ScmpSyscall = __PNR_setresgid32
	SNR_SETRESUID32     ScmpSyscall = __PNR_setresuid32
	SNR_SETREUID32      ScmpSyscall = __PNR_setreuid32
	SNR_SETUID32        ScmpSyscall = __PNR_setuid32
)
