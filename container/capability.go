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

	CAP_SYS_ADMIN    = 0x15
	CAP_SETPCAP      = 0x8
	CAP_DAC_OVERRIDE = 0x1
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
	r, _, errno := syscall.Syscall(
		syscall.SYS_CAPSET,
		uintptr(unsafe.Pointer(hdrp)),
		uintptr(unsafe.Pointer(&datap[0])), 0,
	)
	if r != 0 {
		return errno
	}
	return nil
}

// capBoundingSetDrop drops a capability from the calling thread's capability bounding set.
func capBoundingSetDrop(cap uintptr) error {
	r, _, errno := syscall.Syscall(
		syscall.SYS_PRCTL,
		syscall.PR_CAPBSET_DROP,
		cap, 0,
	)
	if r != 0 {
		return errno
	}
	return nil
}

// capAmbientClearAll clears the ambient capability set of the calling thread.
func capAmbientClearAll() error {
	r, _, errno := syscall.Syscall(
		syscall.SYS_PRCTL,
		PR_CAP_AMBIENT,
		PR_CAP_AMBIENT_CLEAR_ALL, 0,
	)
	if r != 0 {
		return errno
	}
	return nil
}

// capAmbientRaise adds to the ambient capability set of the calling thread.
func capAmbientRaise(cap uintptr) error {
	r, _, errno := syscall.Syscall(
		syscall.SYS_PRCTL,
		PR_CAP_AMBIENT,
		PR_CAP_AMBIENT_RAISE,
		cap,
	)
	if r != 0 {
		return errno
	}
	return nil
}
