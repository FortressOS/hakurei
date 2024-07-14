package main

import "C"
import (
	"errors"
	"unsafe"
)

//#include <stdlib.h>
//#include <xcb/xcb.h>
//#cgo linux LDFLAGS: -lxcb
import "C"

const (
	xcbHostModeInsert = C.XCB_HOST_MODE_INSERT
	xcbHostModeDelete = C.XCB_HOST_MODE_DELETE

	xcbFamilyInternet          = C.XCB_FAMILY_INTERNET
	xcbFamilyDecnet            = C.XCB_FAMILY_DECNET
	xcbFamilyChaos             = C.XCB_FAMILY_CHAOS
	xcbFamilyServerInterpreted = C.XCB_FAMILY_SERVER_INTERPRETED
	xcbFamilyInternet6         = C.XCB_FAMILY_INTERNET_6
)

func changeHosts(mode, family C.uint8_t, address string) error {
	var c *C.xcb_connection_t
	c = C.xcb_connect(nil, nil)
	defer C.xcb_disconnect(c)

	if err := xcbHandleConnectionError(c); err != nil {
		return err
	}

	addr := C.CString(address)
	cookie := C.xcb_change_hosts_checked(c, mode, family, C.ushort(len(address)), (*C.uchar)(unsafe.Pointer(addr)))
	C.free(unsafe.Pointer(addr))

	if err := xcbHandleConnectionError(c); err != nil {
		return err
	}

	e := C.xcb_request_check(c, cookie)
	if e != nil {
		defer C.free(unsafe.Pointer(e))
		return errors.New("xcb_change_hosts() failed")
	}

	return nil
}

func xcbHandleConnectionError(c *C.xcb_connection_t) error {
	if errno := C.xcb_connection_has_error(c); errno != 0 {
		switch errno {
		case C.XCB_CONN_ERROR:
			return errors.New("connection error")
		case C.XCB_CONN_CLOSED_EXT_NOTSUPPORTED:
			return errors.New("extension not supported")
		case C.XCB_CONN_CLOSED_MEM_INSUFFICIENT:
			return errors.New("memory not available")
		case C.XCB_CONN_CLOSED_REQ_LEN_EXCEED:
			return errors.New("request length exceeded")
		case C.XCB_CONN_CLOSED_PARSE_ERR:
			return errors.New("invalid display string")
		case C.XCB_CONN_CLOSED_INVALID_SCREEN:
			return errors.New("server has no screen matching display")
		default:
			return errors.New("generic X11 failure")
		}
	} else {
		return nil
	}
}
