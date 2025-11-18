package pipewire

import (
	"errors"
	"os"
	"syscall"

	"hakurei.app/container/check"
)

// SecurityContext holds resources associated with a PipeWire security context.
type SecurityContext struct {
	// Pipe with its write end passed to the PipeWire security context.
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

	e := Error{RCleanup, sc.bindPath.String(), errors.Join(
		syscall.Close(sc.closeFds[1]),
		syscall.Close(sc.closeFds[0]),
		// there is still technically a TOCTOU here but this is internal
		// and has access to the privileged pipewire socket, so it only
		// receives trusted input (e.g. from cmd/hakurei) anyway
		os.Remove(sc.bindPath.String()),
	)}
	if e.Errno != nil {
		return &e
	}

	return nil
}

// New creates a new security context on the PipeWire remote at remotePath
// or auto-detected, and associates it with a new socket bound to bindPath.
//
// New does not attach a finalizer to the resulting [SecurityContext] struct.
// The caller is responsible for calling [SecurityContext.Close].
//
// A non-nil error unwraps to concrete type [Error].
func New(remotePath, bindPath *check.Absolute) (*SecurityContext, error) {
	// ensure bindPath is available
	if f, err := os.Create(bindPath.String()); err != nil {
		return nil, &Error{RCreate, bindPath.String(), err}
	} else if err = f.Close(); err != nil {
		return nil, &Error{RCreate, bindPath.String(), err}
	} else if err = os.Remove(bindPath.String()); err != nil {
		return nil, &Error{RCreate, bindPath.String(), err}
	}

	// write end passed to PipeWire security context close_fd
	var closeFds [2]int
	if err := syscall.Pipe2(closeFds[0:], syscall.O_CLOEXEC); err != nil {
		return nil, err
	}

	// zero value causes auto-detect
	var remotePathVal string
	if remotePath != nil {
		remotePathVal = remotePath.String()
	}

	// returned error is already wrapped
	if err := securityContextBind(
		bindPath.String(),
		remotePathVal,
		closeFds[1],
	); err != nil {
		return nil, errors.Join(err, // already wrapped
			syscall.Close(closeFds[1]),
			syscall.Close(closeFds[0]),
		)
	} else {
		return &SecurityContext{closeFds, bindPath}, nil
	}
}
