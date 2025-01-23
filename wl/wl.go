package wl

//go:generate sh -c "wayland-scanner client-header `pkg-config --variable=datarootdir wayland-protocols`/wayland-protocols/staging/security-context/security-context-v1.xml security-context-v1-protocol.h"
//go:generate sh -c "wayland-scanner private-code `pkg-config --variable=datarootdir wayland-protocols`/wayland-protocols/staging/security-context/security-context-v1.xml security-context-v1-protocol.c"

/*
#cgo linux pkg-config: --static wayland-client
#cgo freebsd openbsd LDFLAGS: -lwayland-client

#include "wayland-bind.h"
*/
import "C"
import "errors"

var resErr = [...]error{
	0: nil,
	1: errors.New("wl_display_connect_to_fd() failed"),
	2: errors.New("wp_security_context_v1 not available"),
}

func bindWaylandFd(socketPath string, fd uintptr, appID, instanceID string, syncFD uintptr) error {
	res := C.bind_wayland_fd(C.CString(socketPath), C.int(fd), C.CString(appID), C.CString(instanceID), C.int(syncFD))
	return resErr[int32(res)]
}
