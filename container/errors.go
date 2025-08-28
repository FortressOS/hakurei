package container

import (
	"errors"
	"os"
	"syscall"
)

type MountError struct {
	Source, Target, Fstype string

	Flags uintptr
	Data  string
	syscall.Errno
}

func (e *MountError) Unwrap() error {
	if e.Errno == 0 {
		return nil
	}
	return e.Errno
}

func (e *MountError) Error() string {
	if e.Flags&syscall.MS_BIND != 0 {
		if e.Flags&syscall.MS_REMOUNT != 0 {
			return "remount " + e.Target + ": " + e.Errno.Error()
		}
		return "bind " + e.Source + " on " + e.Target + ": " + e.Errno.Error()
	}

	if e.Fstype != FstypeNULL {
		return "mount " + e.Fstype + " on " + e.Target + ": " + e.Errno.Error()
	}

	// fallback case: if this is reached, the conditions for it to occur should be handled above
	return "mount " + e.Target + ": " + e.Errno.Error()
}

// errnoFallback returns the concrete errno from an error, or a [os.PathError] fallback.
func errnoFallback(op, path string, err error) (syscall.Errno, *os.PathError) {
	var errno syscall.Errno
	if !errors.As(err, &errno) {
		return 0, &os.PathError{Op: op, Path: path, Err: err}
	}
	return errno, nil
}

// mount wraps syscall.Mount for error handling.
func mount(source, target, fstype string, flags uintptr, data string) error {
	err := syscall.Mount(source, target, fstype, flags, data)
	if err == nil {
		return nil
	}
	if errno, pathError := errnoFallback("mount", target, err); pathError != nil {
		return pathError
	} else {
		return &MountError{source, target, fstype, flags, data, errno}
	}
}
