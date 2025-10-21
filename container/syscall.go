package container

import (
	. "syscall"
	"unsafe"
)

// Prctl manipulates various aspects of the behavior of the calling thread or process.
func Prctl(op, arg2, arg3 uintptr) error {
	r, _, errno := Syscall(SYS_PRCTL, op, arg2, arg3)
	if r < 0 {
		return errno
	}
	return nil
}

// SetPtracer allows processes to ptrace(2) the calling process.
func SetPtracer(pid uintptr) error { return Prctl(PR_SET_PTRACER, pid, 0) }

// linux/sched/coredump.h
const (
	SUID_DUMP_DISABLE = iota
	SUID_DUMP_USER
)

// SetDumpable sets the "dumpable" attribute of the calling process.
func SetDumpable(dumpable uintptr) error { return Prctl(PR_SET_DUMPABLE, dumpable, 0) }

// SetNoNewPrivs sets the calling thread's no_new_privs attribute.
func SetNoNewPrivs() error { return Prctl(PR_SET_NO_NEW_PRIVS, 1, 0) }

// Isatty tests whether a file descriptor refers to a terminal.
func Isatty(fd int) bool {
	var buf [8]byte
	r, _, _ := Syscall(
		SYS_IOCTL,
		uintptr(fd),
		TIOCGWINSZ,
		uintptr(unsafe.Pointer(&buf[0])),
	)
	return r == 0
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
		if err != EINTR {
			return err
		}
	}
}
