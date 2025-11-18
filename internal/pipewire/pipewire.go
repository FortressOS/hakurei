// Package pipewire implements the client side of PipeWire Security Context interface.
package pipewire

/*
#cgo linux pkg-config: --static libpipewire-0.3

#include "pipewire-helper.h"
#include <pipewire/pipewire.h>
*/
import "C"
import (
	"errors"
	"os"
	"strings"
	"syscall"
)

const (
	// Version is the value of pw_get_headers_version().
	Version = string(byte(C.PW_MAJOR+'0')) + "." + string(byte(C.PW_MINOR+'0')) + "." + string(byte(C.PW_MICRO+'0'))

	// Remote is the environment with the remote name.
	Remote = "PIPEWIRE_REMOTE"
)

type (
	// Res is the outcome of a call to [New].
	Res = C.hakurei_pipewire_res

	// An Error represents a failure during [New].
	Error struct {
		// Where the failure occurred.
		Cause Res
		// Attempted pathname socket.
		Path string
		// Global errno value set during the fault.
		Errno error
	}
)

// withPrefix returns prefix suffixed with errno description if available.
func (e *Error) withPrefix(prefix string) string {
	if e.Errno == nil {
		return prefix
	}
	return prefix + ": " + e.Errno.Error()
}

const (
	// RSuccess is returned on a successful call.
	RSuccess Res = C.HAKUREI_PIPEWIRE_SUCCESS
	// RMainloop is returned if pw_main_loop_new failed. The global errno is set.
	RMainloop Res = C.HAKUREI_PIPEWIRE_MAINLOOP
	// RContext is returned if pw_context_new failed. The global errno is set.
	RContext Res = C.HAKUREI_PIPEWIRE_CTX
	// RConnect is returned if pw_context_connect failed. The global errno is set.
	RConnect Res = C.HAKUREI_PIPEWIRE_CONNECT
	// RRegistry is returned if pw_core_get_registry failed. The global errno is set.
	RRegistry Res = C.HAKUREI_PIPEWIRE_REGISTRY
	// RNotAvail is returned if no security context object found after roundtrip.
	RNotAvail Res = C.HAKUREI_PIPEWIRE_NOT_AVAIL
	// RSocket is returned if socket failed. The global errno is set.
	RSocket Res = C.HAKUREI_PIPEWIRE_SOCKET
	// RBind is returned if bind failed. The global errno is set.
	RBind Res = C.HAKUREI_PIPEWIRE_BIND
	// RListen is returned if listen failed. The global errno is set.
	RListen Res = C.HAKUREI_PIPEWIRE_LISTEN
	// RAttach is returned if pw_security_context_create failed.
	// The internal create_result is translated and set as the global errno.
	RAttach Res = C.HAKUREI_PIPEWIRE_ATTACH

	// RCreate is returned if ensuring pathname availability failed. Returned by [New].
	RCreate Res = C.HAKUREI_PIPEWIRE_CREAT

	// RCleanup is returned if cleanup fails. Returned by [SecurityContext.Close].
	RCleanup Res = C.HAKUREI_PIPEWIRE_CLEANUP
)

func (e *Error) Unwrap() error   { return e.Errno }
func (e *Error) Message() string { return e.Error() }
func (e *Error) Error() string {
	switch e.Cause {
	case RSuccess:
		if e.Errno == nil {
			return "success"
		}
		return e.Errno.Error()

	case RMainloop:
		return e.withPrefix("pw_main_loop_new failed")
	case RContext:
		return e.withPrefix("pw_context_new failed")
	case RConnect:
		return e.withPrefix("pw_context_connect failed")
	case RRegistry:
		return e.withPrefix("pw_core_get_registry failed")
	case RNotAvail:
		return "no security context object found"

	case RSocket:
		if e.Errno == nil {
			return "socket operation failed"
		}
		return "socket: " + e.Errno.Error()
	case RBind:
		return e.withPrefix("cannot bind " + e.Path)
	case RListen:
		return e.withPrefix("cannot listen on " + e.Path)

	case RAttach:
		return e.withPrefix("pw_security_context_create failed")

	case RCreate:
		if e.Errno == nil {
			return "cannot ensure pipewire pathname socket"
		}
		return e.Errno.Error()

	case RCleanup:
		var pathError *os.PathError
		if errors.As(e.Errno, &pathError) && pathError != nil {
			return pathError.Error()
		}

		var errno syscall.Errno
		if errors.As(e.Errno, &errno) && errno != 0 {
			return "cannot close pipewire close_fd pipe: " + errno.Error()
		}

		return e.withPrefix("cannot hang up pipewire security context")

	default:
		return e.withPrefix("impossible outcome") /* not reached */
	}
}

// securityContextBind calls hakurei_pw_security_context_bind.
//
// A non-nil error has concrete type [Error].
func securityContextBind(socketPath, remotePath string, closeFd int) error {
	if hasNull(socketPath) || hasNull(remotePath) {
		return &Error{Cause: RBind, Path: socketPath, Errno: errors.New("argument contains NUL character")}
	}
	if !C.hakurei_pw_is_valid_size_sun_path(C.size_t(len(socketPath))) {
		return &Error{Cause: RBind, Path: socketPath, Errno: errors.New("socket pathname too long")}
	}

	var e Error
	var remotePathP *C.char = nil
	if remotePath != "" {
		remotePathP = C.CString(remotePath)
	}
	e.Cause, e.Errno = C.hakurei_pw_security_context_bind(
		C.CString(socketPath),
		remotePathP,
		C.int(closeFd),
	)
	if e.Cause == RSuccess {
		return nil
	}
	e.Path = socketPath
	return &e
}

// hasNull returns whether s contains the NUL character.
func hasNull(s string) bool { return strings.IndexByte(s, 0) > -1 }
