package seccomp

/*
#cgo linux pkg-config: --static libseccomp

#include <seccomp.h>
*/
import "C"
import "syscall"

const (
	SYS_NEWFSTATAT = syscall.SYS_FSTATAT
)

var syscallNumExtra = map[string]int{
	"uselib":          SYS_USELIB,
	"clock_adjtime64": SYS_CLOCK_ADJTIME64,
	"clock_settime64": SYS_CLOCK_SETTIME64,
	"umount":          SYS_UMOUNT,
	"chown":           SYS_CHOWN,
	"chown32":         SYS_CHOWN32,
	"fchown32":        SYS_FCHOWN32,
	"lchown":          SYS_LCHOWN,
	"lchown32":        SYS_LCHOWN32,
	"setgid32":        SYS_SETGID32,
	"setgroups32":     SYS_SETGROUPS32,
	"setregid32":      SYS_SETREGID32,
	"setresgid32":     SYS_SETRESGID32,
	"setresuid32":     SYS_SETRESUID32,
	"setreuid32":      SYS_SETREUID32,
	"setuid32":        SYS_SETUID32,
	"modify_ldt":      SYS_MODIFY_LDT,
	"subpage_prot":    SYS_SUBPAGE_PROT,
	"switch_endian":   SYS_SWITCH_ENDIAN,
	"vm86":            SYS_VM86,
	"vm86old":         SYS_VM86OLD,
}

const (
	SYS_USELIB          = C.__SNR_uselib
	SYS_CLOCK_ADJTIME64 = C.__SNR_clock_adjtime64
	SYS_CLOCK_SETTIME64 = C.__SNR_clock_settime64
	SYS_UMOUNT          = C.__SNR_umount
	SYS_CHOWN           = C.__SNR_chown
	SYS_CHOWN32         = C.__SNR_chown32
	SYS_FCHOWN32        = C.__SNR_fchown32
	SYS_LCHOWN          = C.__SNR_lchown
	SYS_LCHOWN32        = C.__SNR_lchown32
	SYS_SETGID32        = C.__SNR_setgid32
	SYS_SETGROUPS32     = C.__SNR_setgroups32
	SYS_SETREGID32      = C.__SNR_setregid32
	SYS_SETRESGID32     = C.__SNR_setresgid32
	SYS_SETRESUID32     = C.__SNR_setresuid32
	SYS_SETREUID32      = C.__SNR_setreuid32
	SYS_SETUID32        = C.__SNR_setuid32
	SYS_MODIFY_LDT      = C.__SNR_modify_ldt
	SYS_SUBPAGE_PROT    = C.__SNR_subpage_prot
	SYS_SWITCH_ENDIAN   = C.__SNR_switch_endian
	SYS_VM86            = C.__SNR_vm86
	SYS_VM86OLD         = C.__SNR_vm86old
)
