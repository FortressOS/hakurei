package sandbox

import (
	"os"
	"syscall"
)

/*
#include <sys/quota.h>
*/
import "C"

const NULL = 0

func TrySyscalls() error {
	testCases := []struct {
		name  string
		errno syscall.Errno

		trap, a1, a2, a3, a4, a5, a6 uintptr
	}{
		{"syslog", syscall.EPERM, syscall.SYS_SYSLOG, 0, NULL, NULL, NULL, NULL, NULL},
		{"uselib", syscall.EPERM, syscall.SYS_USELIB, 0, NULL, NULL, NULL, NULL, NULL},
		{"acct", syscall.EPERM, syscall.SYS_ACCT, 0, NULL, NULL, NULL, NULL, NULL},
		{"quotactl", syscall.EPERM, syscall.SYS_QUOTACTL, C.Q_GETQUOTA, NULL, uintptr(os.Getuid()), NULL, NULL, NULL},
		{"add_key", syscall.EPERM, syscall.SYS_ADD_KEY, NULL, NULL, NULL, NULL, NULL, NULL},
		{"keyctl", syscall.EPERM, syscall.SYS_KEYCTL, NULL, NULL, NULL, NULL, NULL, NULL},
		{"request_key", syscall.EPERM, syscall.SYS_REQUEST_KEY, NULL, NULL, NULL, NULL, NULL, NULL},
		{"move_pages", syscall.EPERM, syscall.SYS_MOVE_PAGES, uintptr(os.Getpid()), NULL, NULL, NULL, NULL, NULL},
		{"mbind", syscall.EPERM, syscall.SYS_MBIND, NULL, NULL, NULL, NULL, NULL, NULL},
		{"get_mempolicy", syscall.EPERM, syscall.SYS_GET_MEMPOLICY, NULL, NULL, NULL, NULL, NULL, NULL},
		{"set_mempolicy", syscall.EPERM, syscall.SYS_SET_MEMPOLICY, NULL, NULL, NULL, NULL, NULL, NULL},
		{"migrate_pages", syscall.EPERM, syscall.SYS_MIGRATE_PAGES, NULL, NULL, NULL, NULL, NULL, NULL},
	}

	for _, tc := range testCases {
		if _, _, errno := syscall.Syscall6(tc.trap, tc.a1, tc.a2, tc.a3, tc.a4, tc.a5, tc.a6); errno != tc.errno {
			printf("[FAIL] %s: %v, want %v", tc.name, errno, tc.errno)
			return errno
		}
		printf("[ OK ] %s: %v", tc.name, tc.errno)
	}

	return nil
}
