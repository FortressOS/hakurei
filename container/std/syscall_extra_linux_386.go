package std

var syscallNumExtra = map[string]int{
	"kexec_file_load": SYS_KEXEC_FILE_LOAD,
	"subpage_prot":    SYS_SUBPAGE_PROT,
	"switch_endian":   SYS_SWITCH_ENDIAN,
}

const (
	SYS_KEXEC_FILE_LOAD = __PNR_kexec_file_load
	SYS_SUBPAGE_PROT    = __PNR_subpage_prot
	SYS_SWITCH_ENDIAN   = __PNR_switch_endian
)
