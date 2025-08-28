package container

import (
	"errors"
	"os"
	"reflect"
	"syscall"
	"testing"
)

func TestMountError(t *testing.T) {
	testCases := []struct {
		name  string
		err   error
		errno syscall.Errno
		want  string
	}{
		{"bind", &MountError{
			Source: "/host/nix/store",
			Target: "/sysroot/nix/store",
			Fstype: FstypeNULL,
			Flags:  syscall.MS_SILENT | syscall.MS_BIND | syscall.MS_REC,
			Data:   zeroString,
			Errno:  syscall.ENOSYS,
		}, syscall.ENOSYS,
			"bind /host/nix/store on /sysroot/nix/store: function not implemented"},

		{"remount", &MountError{
			Source: SourceNone,
			Target: "/sysroot/nix/store",
			Fstype: FstypeNULL,
			Flags:  syscall.MS_SILENT | syscall.MS_BIND | syscall.MS_REMOUNT,
			Data:   zeroString,
			Errno:  syscall.EPERM,
		}, syscall.EPERM,
			"remount /sysroot/nix/store: operation not permitted"},

		{"overlay", &MountError{
			Source: SourceOverlay,
			Target: sysrootPath,
			Fstype: FstypeOverlay,
			Data:   `lowerdir=/host/var/lib/planterette/base/debian\:f92c9052`,
			Errno:  syscall.EINVAL,
		}, syscall.EINVAL,
			"mount overlay on /sysroot: invalid argument"},

		{"fallback", &MountError{
			Source: SourceNone,
			Target: sysrootPath,
			Fstype: FstypeNULL,
			Errno:  syscall.ENOTRECOVERABLE,
		}, syscall.ENOTRECOVERABLE,
			"mount /sysroot: state not recoverable"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("is", func(t *testing.T) {
				if !errors.Is(tc.err, tc.errno) {
					t.Errorf("Is: %#v is not %v", tc.err, tc.errno)
				}
			})
			t.Run("error", func(t *testing.T) {
				if got := tc.err.Error(); got != tc.want {
					t.Errorf("Error: %q, want %q", got, tc.want)
				}
			})
		})
	}

	t.Run("zero", func(t *testing.T) {
		if errors.Is(new(MountError), syscall.Errno(0)) {
			t.Errorf("Is: zero MountError unexpected true")
		}
	})
}

func TestErrnoFallback(t *testing.T) {
	testCases := []struct {
		name      string
		err       error
		wantErrno syscall.Errno
		wantPath  *os.PathError
	}{
		{"mount", &MountError{
			Errno: syscall.ENOTRECOVERABLE,
		}, syscall.ENOTRECOVERABLE, nil},

		{"path errno", &os.PathError{
			Err: syscall.ETIMEDOUT,
		}, syscall.ETIMEDOUT, nil},

		{"fallback", errUnique, 0, &os.PathError{
			Op:   "fallback",
			Path: "/proc/nonexistent",
			Err:  errUnique,
		}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errno, err := errnoFallback(tc.name, Nonexistent, tc.err)
			if errno != tc.wantErrno {
				t.Errorf("errnoFallback: errno = %v, want %v", errno, tc.wantErrno)
			}
			if !reflect.DeepEqual(err, tc.wantPath) {
				t.Errorf("errnoFallback: pathError = %#v, want %#v", err, tc.wantPath)
			}
		})
	}
}
