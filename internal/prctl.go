package internal

import "syscall"

func PR_SET_DUMPABLE__SUID_DUMP_DISABLE() error {
	// linux/sched/coredump.h
	if _, _, errno := syscall.RawSyscall(syscall.SYS_PRCTL, syscall.PR_SET_DUMPABLE, 0, 0); errno != 0 {
		return errno
	}

	return nil
}

func PR_SET_PDEATHSIG__SIGKILL() error {
	if _, _, errno := syscall.RawSyscall(syscall.SYS_PRCTL, syscall.PR_SET_PDEATHSIG, uintptr(syscall.SIGKILL), 0); errno != 0 {
		return errno
	}

	return nil
}
