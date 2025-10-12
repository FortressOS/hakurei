package seccomp_test

import (
	"errors"
	"runtime"
	"syscall"
	"testing"

	"hakurei.app/container/seccomp"
)

func TestLibraryError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		sample  *seccomp.LibraryError
		want    string
		wantIs  bool
		compare error
	}{
		{
			"full",
			&seccomp.LibraryError{Prefix: "seccomp_export_bpf failed", Seccomp: syscall.ECANCELED, Errno: syscall.EBADF},
			"seccomp_export_bpf failed: operation canceled (bad file descriptor)",
			true,
			&seccomp.LibraryError{Prefix: "seccomp_export_bpf failed", Seccomp: syscall.ECANCELED, Errno: syscall.EBADF},
		},
		{
			"errno only",
			&seccomp.LibraryError{Prefix: "seccomp_init failed", Errno: syscall.ENOMEM},
			"seccomp_init failed: cannot allocate memory",
			false,
			nil,
		},
		{
			"seccomp only",
			&seccomp.LibraryError{Prefix: "internal libseccomp failure", Seccomp: syscall.EFAULT},
			"internal libseccomp failure: bad address",
			true,
			syscall.EFAULT,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if errors.Is(tc.sample, tc.compare) != tc.wantIs {
				t.Errorf("errors.Is(%#v, %#v) did not return %v",
					tc.sample, tc.compare, tc.wantIs)
			}

			if got := tc.sample.Error(); got != tc.want {
				t.Errorf("Error: %q, want %q",
					got, tc.want)
			}
		})
	}

	t.Run("invalid", func(t *testing.T) {
		t.Parallel()

		wantPanic := "invalid libseccomp error"
		defer func() {
			if r := recover(); r != wantPanic {
				t.Errorf("panic: %q, want %q", r, wantPanic)
			}
		}()
		runtime.KeepAlive(new(seccomp.LibraryError).Error())
	})
}
