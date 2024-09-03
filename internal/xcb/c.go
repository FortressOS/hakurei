package xcb

import (
	"errors"
)

//#include <stdlib.h>
//#include <xcb/xcb.h>
//#cgo linux LDFLAGS: -lxcb
import "C"

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
