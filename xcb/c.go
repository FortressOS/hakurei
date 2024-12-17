package xcb

import (
	"runtime"
	"unsafe"
)

/*
#cgo linux pkg-config: xcb

#include <stdlib.h>
#include <xcb/xcb.h>

static int _go_xcb_change_hosts_checked(xcb_connection_t *c, uint8_t mode, uint8_t family, uint16_t address_len, const uint8_t *address) {
	xcb_void_cookie_t cookie = xcb_change_hosts_checked(c, mode, family, address_len, address);
	free((void *)address);

	int errno = xcb_connection_has_error(c);
	if (errno != 0)
		return errno;

	xcb_generic_error_t *e = xcb_request_check(c, cookie);
	if (e != NULL) {
		// don't want to deal with xcb errors
		free((void *)e);
		return -1;
	}

	return 0;
}
*/
import "C"

const (
	HostModeInsert = C.XCB_HOST_MODE_INSERT
	HostModeDelete = C.XCB_HOST_MODE_DELETE

	FamilyInternet          = C.XCB_FAMILY_INTERNET
	FamilyDecnet            = C.XCB_FAMILY_DECNET
	FamilyChaos             = C.XCB_FAMILY_CHAOS
	FamilyServerInterpreted = C.XCB_FAMILY_SERVER_INTERPRETED
	FamilyInternet6         = C.XCB_FAMILY_INTERNET_6
)

type (
	HostMode = C.xcb_host_mode_t
	Family   = C.xcb_family_t
)

func (conn *connection) changeHostsChecked(mode HostMode, family Family, address string) error {
	errno := C._go_xcb_change_hosts_checked(
		conn.c,
		C.uint8_t(mode),
		C.uint8_t(family),
		C.uint16_t(len(address)),
		(*C.uint8_t)(unsafe.Pointer(C.CString(address))),
	)
	switch errno {
	case 0:
		return nil
	case -1:
		return ErrChangeHosts
	default:
		return &ConnectionError{errno}
	}
}

type connection struct{ c *C.xcb_connection_t }

func connect() (*connection, error) {
	conn := newConnection(C.xcb_connect(nil, nil))
	return conn, conn.hasError()
}

func newConnection(c *C.xcb_connection_t) *connection {
	conn := &connection{c}
	runtime.SetFinalizer(conn, (*connection).disconnect)
	return conn
}

const (
	ConnError                 = C.XCB_CONN_ERROR
	ConnClosedExtNotSupported = C.XCB_CONN_CLOSED_EXT_NOTSUPPORTED
	ConnClosedMemInsufficient = C.XCB_CONN_CLOSED_MEM_INSUFFICIENT
	ConnClosedReqLenExceed    = C.XCB_CONN_CLOSED_REQ_LEN_EXCEED
	ConnClosedParseErr        = C.XCB_CONN_CLOSED_PARSE_ERR
	ConnClosedInvalidScreen   = C.XCB_CONN_CLOSED_INVALID_SCREEN
)

type ConnectionError struct{ errno C.int }

func (ce *ConnectionError) Error() string {
	switch ce.errno {
	case ConnError:
		return "connection error"
	case ConnClosedExtNotSupported:
		return "extension not supported"
	case ConnClosedMemInsufficient:
		return "memory not available"
	case ConnClosedReqLenExceed:
		return "request length exceeded"
	case ConnClosedParseErr:
		return "invalid display string"
	case ConnClosedInvalidScreen:
		return "server has no screen matching display"
	default:
		return "generic X11 failure"
	}
}

func (conn *connection) hasError() error {
	errno := C.xcb_connection_has_error(conn.c)
	if errno == 0 {
		return nil
	}
	return &ConnectionError{errno}
}

func (conn *connection) disconnect() {
	C.xcb_disconnect(conn.c)

	// no need for a finalizer anymore
	runtime.SetFinalizer(conn, nil)
}
