package seccomp

/*
#cgo linux pkg-config: --static libseccomp

#include <seccomp.h>
*/
import "C"

var syscallNumExtra = map[string]int{
	"umount": SYS_UMOUNT,
}

const (
	SYS_UMOUNT = C.__PNR_umount
)
