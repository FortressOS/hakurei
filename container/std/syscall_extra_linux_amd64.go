package std

var syscallNumExtra = map[string]int{
	"umount":          SYS_UMOUNT,
	"subpage_prot":    SYS_SUBPAGE_PROT,
	"switch_endian":   SYS_SWITCH_ENDIAN,
	"vm86":            SYS_VM86,
	"vm86old":         SYS_VM86OLD,
	"clock_adjtime64": SYS_CLOCK_ADJTIME64,
	"clock_settime64": SYS_CLOCK_SETTIME64,
	"chown32":         SYS_CHOWN32,
	"fchown32":        SYS_FCHOWN32,
	"lchown32":        SYS_LCHOWN32,
	"setgid32":        SYS_SETGID32,
	"setgroups32":     SYS_SETGROUPS32,
	"setregid32":      SYS_SETREGID32,
	"setresgid32":     SYS_SETRESGID32,
	"setresuid32":     SYS_SETRESUID32,
	"setreuid32":      SYS_SETREUID32,
	"setuid32":        SYS_SETUID32,
}

const (
	SYS_UMOUNT          = __PNR_umount
	SYS_SUBPAGE_PROT    = __PNR_subpage_prot
	SYS_SWITCH_ENDIAN   = __PNR_switch_endian
	SYS_VM86            = __PNR_vm86
	SYS_VM86OLD         = __PNR_vm86old
	SYS_CLOCK_ADJTIME64 = __PNR_clock_adjtime64
	SYS_CLOCK_SETTIME64 = __PNR_clock_settime64
	SYS_CHOWN32         = __PNR_chown32
	SYS_FCHOWN32        = __PNR_fchown32
	SYS_LCHOWN32        = __PNR_lchown32
	SYS_SETGID32        = __PNR_setgid32
	SYS_SETGROUPS32     = __PNR_setgroups32
	SYS_SETREGID32      = __PNR_setregid32
	SYS_SETRESGID32     = __PNR_setresgid32
	SYS_SETRESUID32     = __PNR_setresuid32
	SYS_SETREUID32      = __PNR_setreuid32
	SYS_SETUID32        = __PNR_setuid32
)
