package sandbox

import "syscall"

const (
	O_PATH = 0x200000

	PR_SET_NO_NEW_PRIVS = 0x26

	PR_CAP_AMBIENT           = 47
	PR_CAP_AMBIENT_CLEAR_ALL = 4

	CAP_SYS_ADMIN = 0x15
	CAP_SETPCAP   = 8
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
