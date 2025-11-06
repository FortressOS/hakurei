package std

var syscallNumExtra = map[string]ScmpSyscall{
	"kexec_file_load": SNR_KEXEC_FILE_LOAD,
	"subpage_prot":    SNR_SUBPAGE_PROT,
	"switch_endian":   SNR_SWITCH_ENDIAN,
}

const (
	SNR_KEXEC_FILE_LOAD ScmpSyscall = __PNR_kexec_file_load
	SNR_SUBPAGE_PROT    ScmpSyscall = __PNR_subpage_prot
	SNR_SWITCH_ENDIAN   ScmpSyscall = __PNR_switch_endian
)
