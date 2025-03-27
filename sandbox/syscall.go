package sandbox

import (
	"syscall"
	"unsafe"
)

const (
	O_PATH = 0x200000

	PR_SET_NO_NEW_PRIVS = 0x26

	CAP_SYS_ADMIN = 0x15
	CAP_SETPCAP   = 0x8
)

const (
	SUID_DUMP_DISABLE = iota
	SUID_DUMP_USER
)

func SetDumpable(dumpable uintptr) error {
	// linux/sched/coredump.h
	if _, _, errno := syscall.Syscall(syscall.SYS_PRCTL, syscall.PR_SET_DUMPABLE, dumpable, 0); errno != 0 {
		return errno
	}

	return nil
}

const (
	_LINUX_CAPABILITY_VERSION_3 = 0x20080522

	PR_CAP_AMBIENT           = 0x2f
	PR_CAP_AMBIENT_RAISE     = 0x2
	PR_CAP_AMBIENT_CLEAR_ALL = 0x4
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

// IgnoringEINTR makes a function call and repeats it if it returns an
// EINTR error. This appears to be required even though we install all
// signal handlers with SA_RESTART: see #22838, #38033, #38836, #40846.
// Also #20400 and #36644 are issues in which a signal handler is
// installed without setting SA_RESTART. None of these are the common case,
// but there are enough of them that it seems that we can't avoid
// an EINTR loop.
func IgnoringEINTR(fn func() error) error {
	for {
		err := fn()
		if err != syscall.EINTR {
			return err
		}
	}
}
