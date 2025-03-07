package internal

import "syscall"

const (
	SUID_DUMP_DISABLE = iota
	SUID_DUMP_USER
)

func SetDumpable(dumpable uintptr) error {
	// linux/sched/coredump.h
	if _, _, errno := syscall.RawSyscall(syscall.SYS_PRCTL, syscall.PR_SET_DUMPABLE, dumpable, 0); errno != 0 {
		return errno
	}

	return nil
}

func SetPdeathsig(sig syscall.Signal) error {
	if _, _, errno := syscall.RawSyscall(syscall.SYS_PRCTL, syscall.PR_SET_PDEATHSIG, uintptr(sig), 0); errno != 0 {
		return errno
	}

	return nil
}
