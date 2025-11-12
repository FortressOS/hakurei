// Package xcb implements X11 ChangeHosts via libxcb.
package xcb

import (
	"runtime"
	"unsafe"
)

/*
#cgo linux pkg-config: --static xcb

#include <stdlib.h>
#include <xcb/xcb.h>

static int hakurei_xcb_change_hosts_checked(xcb_connection_t *c,
                                            uint8_t mode, uint8_t family,
                                            uint16_t address_len, const uint8_t *address) {
  int ret;
  xcb_generic_error_t *e;
  xcb_void_cookie_t cookie;

  cookie = xcb_change_hosts_checked(c, mode, family, address_len, address);
  free((void *)address);

  ret = xcb_connection_has_error(c);
  if (ret != 0)
    return ret;

  e = xcb_request_check(c, cookie);
  if (e != NULL) {
    // don't want to deal with xcb errors
    free((void *)e);
    ret = -1;
  }

  return ret;
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
	ret := C.hakurei_xcb_change_hosts_checked(
		conn.c,
		C.uint8_t(mode),
		C.uint8_t(family),
		C.uint16_t(len(address)),
		(*C.uint8_t)(unsafe.Pointer(C.CString(address))),
	)
	switch ret {
	case 0:
		return nil
	case -1:
		return ErrChangeHosts
	default:
		return ConnectionError(ret)
	}
}

type connection struct{ c *C.xcb_connection_t }

func (conn *connection) connect() error {
	conn.c = C.xcb_connect(nil, nil)
	runtime.SetFinalizer(conn, (*connection).disconnect)
	return conn.hasError()
}

func (conn *connection) hasError() error {
	ret := C.xcb_connection_has_error(conn.c)
	if ret == 0 {
		return nil
	}
	return ConnectionError(ret)
}

func (conn *connection) disconnect() {
	C.xcb_disconnect(conn.c)

	// no need for a finalizer anymore
	runtime.SetFinalizer(conn, nil)
}

const (
	ConnError                 ConnectionError = C.XCB_CONN_ERROR
	ConnClosedExtNotSupported ConnectionError = C.XCB_CONN_CLOSED_EXT_NOTSUPPORTED
	ConnClosedMemInsufficient ConnectionError = C.XCB_CONN_CLOSED_MEM_INSUFFICIENT
	ConnClosedReqLenExceed    ConnectionError = C.XCB_CONN_CLOSED_REQ_LEN_EXCEED
	ConnClosedParseErr        ConnectionError = C.XCB_CONN_CLOSED_PARSE_ERR
	ConnClosedInvalidScreen   ConnectionError = C.XCB_CONN_CLOSED_INVALID_SCREEN
)

// ConnectionError represents an error returned by xcb_connection_has_error.
type ConnectionError int

func (ce ConnectionError) Error() string {
	switch ce {
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
