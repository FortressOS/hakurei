package wayland

import (
	"errors"
	"os"
	"syscall"

	"hakurei.app/container/check"
)

// SecurityContext holds resources associated with a Wayland security_context.
type SecurityContext struct {
	// Pipe with its write end passed to security-context-v1.
	closeFds [2]int
}

// Close releases any resources held by [SecurityContext], and prevents further
// connections to its associated socket.
func (sc *SecurityContext) Close() error {
	if sc == nil {
		return os.ErrInvalid
	}
	return errors.Join(
		syscall.Close(sc.closeFds[1]),
		syscall.Close(sc.closeFds[0]),
	)
}

// New creates a new security context on the Wayland display at displayPath
// and associates it with a new socket bound to bindPath.
//
// New does not attach a finalizer to the resulting [SecurityContext] struct.
// The caller is responsible for calling [SecurityContext.Close].
//
// A non-nil error unwraps to concrete type [Error].
func New(displayPath, bindPath *check.Absolute, appID, instanceID string) (*SecurityContext, error) {
	// ensure bindPath is available
	if f, err := os.Create(bindPath.String()); err != nil {
		return nil, &Error{Cause: RHostCreate, Errno: err}
	} else if err = f.Close(); err != nil {
		return nil, &Error{Cause: RHostCreate, Errno: err}
	} else if err = os.Remove(bindPath.String()); err != nil {
		return nil, &Error{Cause: RHostCreate, Errno: err}
	}

	if fd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_STREAM|syscall.SOCK_CLOEXEC, 0); err != nil {
		return nil, &Error{RHostSocket, err}
	} else if err = syscall.Connect(fd, &syscall.SockaddrUnix{Name: displayPath.String()}); err != nil {
		_ = syscall.Close(fd)
		return nil, &Error{RHostConnect, err}
	} else {
		closeFds, bindErr := bindSecurityContext(fd, bindPath, appID, instanceID)
		if bindErr != nil {
			// do not leak the pipe and socket
			err = errors.Join(bindErr, // already wrapped
				syscall.Close(closeFds[1]),
				syscall.Close(closeFds[0]),
				syscall.Close(fd),
			)
		}
		return &SecurityContext{closeFds}, err
	}
}

// bindSecurityContext binds a socket associated to a security context created on serverFd,
// returning the pipe file descriptors used for security-context-v1 close_fd.
//
// A non-nil error unwraps to concrete type [Error].
func bindSecurityContext(serverFd int, bindPath *check.Absolute, appID, instanceID string) ([2]int, error) {
	// write end passed to security-context-v1 close_fd
	var closeFds [2]int
	if err := syscall.Pipe2(closeFds[0:], syscall.O_CLOEXEC); err != nil {
		return closeFds, err
	}

	// returned error is already wrapped
	if err := bindWaylandFd(bindPath.String(), uintptr(serverFd), appID, instanceID, uintptr(closeFds[1])); err != nil {
		return closeFds, errors.Join(err,
			syscall.Close(closeFds[1]),
			syscall.Close(closeFds[0]),
		)
	} else {
		return closeFds, nil
	}
}
