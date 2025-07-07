package container

import (
	"syscall"
	"unsafe"
)

const (
	_LINUX_CAPABILITY_VERSION_3 = 0x20080522

	PR_CAP_AMBIENT           = 0x2f
	PR_CAP_AMBIENT_RAISE     = 0x2
	PR_CAP_AMBIENT_CLEAR_ALL = 0x4

	CAP_SYS_ADMIN = 0x15
	CAP_SETPCAP   = 0x8
)

type (
	capHeader struct {
		version uint32
		pid     int32
	}

	capData struct {
		effective   uint32
		permitted   uint32
		inheritable uint32
	}
)

// See CAP_TO_INDEX in linux/capability.h:
func capToIndex(cap uintptr) uintptr { return cap >> 5 }

// See CAP_TO_MASK in linux/capability.h:
func capToMask(cap uintptr) uint32 { return 1 << uint(cap&31) }

func capset(hdrp *capHeader, datap *[2]capData) error {
	if _, _, errno := syscall.Syscall(syscall.SYS_CAPSET,
		uintptr(unsafe.Pointer(hdrp)),
		uintptr(unsafe.Pointer(&datap[0])), 0); errno != 0 {
		return errno
	}
	return nil
}
