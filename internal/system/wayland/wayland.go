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
	"errors"
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

var resErr = [...]error{
	0: nil,
	1: errors.New("wl_display_connect_to_fd() failed"),
	2: errors.New("wp_security_context_v1 not available"),
}

func bindWaylandFd(socketPath string, fd uintptr, appID, instanceID string, syncFd uintptr) error {
	if hasNull(appID) || hasNull(instanceID) {
		return syscall.EINVAL
	}
	res := C.hakurei_bind_wayland_fd(C.CString(socketPath), C.int(fd), C.CString(appID), C.CString(instanceID), C.int(syncFd))
	return resErr[int32(res)]
}

func hasNull(s string) bool { return strings.IndexByte(s, 0) > -1 }
