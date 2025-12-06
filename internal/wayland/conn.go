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
	// Absolute pathname the socket was bound to.
	bindPath *check.Absolute
}

// Close releases any resources held by [SecurityContext], and prevents further
// connections to its associated socket.
//
// A non-nil error has the concrete type [Error].
func (sc *SecurityContext) Close() error {
	if sc == nil || sc.bindPath == nil {
		return os.ErrInvalid
	}

	e := Error{RCleanup, sc.bindPath.String(), "", errors.Join(
		syscall.Close(sc.closeFds[1]),
		syscall.Close(sc.closeFds[0]),
		// there is still technically a TOCTOU here but this is internal
		// and has access to the privileged wayland socket, so it only
		// receives trusted input (e.g. from cmd/hakurei) anyway
		os.Remove(sc.bindPath.String()),
	)}
	if e.Errno != nil {
		return &e
	}

	return nil
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
		return nil, &Error{RCreate, bindPath.String(), displayPath.String(), err}
	} else if err = f.Close(); err != nil {
		_ = os.Remove(bindPath.String())
		return nil, &Error{RCreate, bindPath.String(), displayPath.String(), err}
	} else if err = os.Remove(bindPath.String()); err != nil {
		return nil, &Error{RCreate, bindPath.String(), displayPath.String(), err}
	}

	if fd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_STREAM|syscall.SOCK_CLOEXEC, 0); err != nil {
		return nil, &Error{RHostSocket, bindPath.String(), displayPath.String(), err}
	} else if err = syscall.Connect(fd, &syscall.SockaddrUnix{Name: displayPath.String()}); err != nil {
		_ = syscall.Close(fd)
		return nil, &Error{RHostConnect, bindPath.String(), displayPath.String(), err}
	} else {
		closeFds, bindErr := securityContextBindPipe(fd, bindPath, appID, instanceID)
		if bindErr != nil {
			// securityContextBindPipe does not try to remove the socket during cleanup
			closeErr := os.Remove(bindPath.String())
			if closeErr != nil && errors.Is(closeErr, os.ErrNotExist) {
				closeErr = nil
			}

			err = errors.Join(bindErr, // already wrapped
				closeErr,
				// do not leak the socket
				syscall.Close(fd),
			)
		}
		return &SecurityContext{closeFds, bindPath}, err
	}
}

// securityContextBindPipe binds a socket associated to a security context created on serverFd,
// returning the pipe file descriptors used for security-context-v1 close_fd.
//
// A non-nil error unwraps to concrete type [Error].
func securityContextBindPipe(
	serverFd int,
	bindPath *check.Absolute,
	appID, instanceID string,
) ([2]int, error) {
	// write end passed to security-context-v1 close_fd
	var closeFds [2]int
	if err := syscall.Pipe2(closeFds[0:], syscall.O_CLOEXEC); err != nil {
		return closeFds, err
	}

	// returned error is already wrapped
	if err := securityContextBind(
		bindPath.String(),
		serverFd,
		appID, instanceID,
		closeFds[1],
	); err != nil {
		return closeFds, errors.Join(err,
			syscall.Close(closeFds[1]),
			syscall.Close(closeFds[0]),
		)
	} else {
		return closeFds, nil
	}
}
