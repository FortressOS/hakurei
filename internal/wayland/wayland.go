// Package wayland implements Wayland security_context_v1 protocol.
package wayland

//go:generate sh -c "wayland-scanner client-header `pkg-config --variable=datarootdir wayland-protocols`/wayland-protocols/staging/security-context/security-context-v1.xml security-context-v1-protocol.h"
//go:generate sh -c "wayland-scanner private-code `pkg-config --variable=datarootdir wayland-protocols`/wayland-protocols/staging/security-context/security-context-v1.xml security-context-v1-protocol.c"

/*
#cgo linux pkg-config: --static wayland-client
#cgo freebsd openbsd LDFLAGS: -lwayland-client

#include "wayland-client-helper.h"
*/
import "C"
import (
	"strings"
	"syscall"
)

const (
	// Display contains the name of the server socket
	// (https://gitlab.freedesktop.org/wayland/wayland/-/blob/1.23.1/src/wayland-client.c#L1147)
	// which is concatenated with XDG_RUNTIME_DIR
	// (https://gitlab.freedesktop.org/wayland/wayland/-/blob/1.23.1/src/wayland-client.c#L1171)
	// or used as-is if absolute
	// (https://gitlab.freedesktop.org/wayland/wayland/-/blob/1.23.1/src/wayland-client.c#L1176).
	Display = "WAYLAND_DISPLAY"

	// FallbackName is used as the wayland socket name if WAYLAND_DISPLAY is unset
	// (https://gitlab.freedesktop.org/wayland/wayland/-/blob/1.23.1/src/wayland-client.c#L1149).
	FallbackName = "wayland-0"
)

type (
	// Res is the outcome of a call to [New].
	Res = C.hakurei_wayland_res

	// An Error represents a failure during [New].
	Error struct {
		// Where the failure occurred.
		Cause Res
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
	RSuccess Res = C.HAKUREI_WAYLAND_SUCCESS
	// RConnect is returned if wl_display_connect_to_fd failed. The global errno is set.
	RConnect Res = C.HAKUREI_WAYLAND_CONNECT
	// RListener is returned if wl_registry_add_listener failed. The global errno is set.
	RListener Res = C.HAKUREI_WAYLAND_LISTENER
	// RRoundtrip is returned if wl_display_roundtrip failed. The global errno is set.
	RRoundtrip Res = C.HAKUREI_WAYLAND_ROUNDTRIP
	// RNotAvail is returned if compositor does not implement wp_security_context_v1.
	RNotAvail Res = C.HAKUREI_WAYLAND_NOT_AVAIL
	// RSocket is returned if socket failed. The global errno is set.
	RSocket Res = C.HAKUREI_WAYLAND_SOCKET
	// RBind is returned if bind failed. The global errno is set.
	RBind Res = C.HAKUREI_WAYLAND_BIND
	// RListen is returned if listen failed. The global errno is set.
	RListen Res = C.HAKUREI_WAYLAND_LISTEN

	// RHostCreate is returned if ensuring pathname availability failed. Returned by [New].
	RHostCreate Res = C.HAKUREI_WAYLAND_HOST_CREAT
	// RHostSocket is returned if socket failed for host server. Returned by [New].
	RHostSocket Res = C.HAKUREI_WAYLAND_HOST_SOCKET
	// RHostConnect is returned if connect failed for host server. Returned by [New].
	RHostConnect Res = C.HAKUREI_WAYLAND_HOST_CONNECT
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

	case RConnect:
		return e.withPrefix("wl_display_connect_to_fd failed")
	case RListener:
		return e.withPrefix("wl_registry_add_listener failed")
	case RRoundtrip:
		return e.withPrefix("wl_display_roundtrip failed")
	case RNotAvail:
		return "compositor does not implement security_context_v1"

	case RSocket, RBind, RListen:
		if e.Errno == nil {
			return "socket operation failed"
		}
		return e.Errno.Error()

	case RHostCreate:
		if e.Errno == nil {
			return "cannot ensure wayland pathname socket"
		}
		return e.Errno.Error()
	case RHostSocket:
		return e.withPrefix("socket for host wayland server")
	case RHostConnect:
		return e.withPrefix("connect to host wayland server")

	default:
		return e.withPrefix("impossible outcome") /* not reached */
	}
}

// bindWaylandFd calls hakurei_bind_wayland_fd. A non-nil error has concrete type [Error].
func bindWaylandFd(socketPath string, fd uintptr, appID, instanceID string, syncFd uintptr) error {
	if hasNull(appID) || hasNull(instanceID) {
		return syscall.EINVAL
	}

	var e Error
	e.Cause, e.Errno = C.hakurei_bind_wayland_fd(
		C.CString(socketPath),
		C.int(fd),
		C.CString(appID),
		C.CString(instanceID),
		C.int(syncFd),
	)

	if e.Cause == RSuccess {
		return nil
	}
	return &e
}

// hasNull returns whether s contains the NUL character.
func hasNull(s string) bool { return strings.IndexByte(s, 0) > -1 }
