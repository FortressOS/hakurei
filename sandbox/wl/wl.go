package wl

//go:generate sh -c "wayland-scanner client-header `pkg-config --variable=datarootdir wayland-protocols`/wayland-protocols/staging/security-context/security-context-v1.xml security-context-v1-protocol.h"
//go:generate sh -c "wayland-scanner private-code `pkg-config --variable=datarootdir wayland-protocols`/wayland-protocols/staging/security-context/security-context-v1.xml security-context-v1-protocol.c"

/*
#cgo linux pkg-config: --static wayland-client
#cgo freebsd openbsd LDFLAGS: -lwayland-client

#include "wayland-bind.h"
*/
import "C"
import (
	"errors"
	"strings"
)

var (
	ErrContainsNull = errors.New("string contains null character")
)

var resErr = [...]error{
	0: nil,
	1: errors.New("wl_display_connect_to_fd() failed"),
	2: errors.New("wp_security_context_v1 not available"),
}

func bindWaylandFd(socketPath string, fd uintptr, appID, instanceID string, syncFd uintptr) error {
	if hasNull(appID) || hasNull(instanceID) {
		return ErrContainsNull
	}
	res := C.hakurei_bind_wayland_fd(C.CString(socketPath), C.int(fd), C.CString(appID), C.CString(instanceID), C.int(syncFd))
	return resErr[int32(res)]
}

func hasNull(s string) bool { return strings.IndexByte(s, '\x00') > -1 }
