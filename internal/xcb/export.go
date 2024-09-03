package xcb

//#include <stdlib.h>
//#include <xcb/xcb.h>
//#cgo linux LDFLAGS: -lxcb
import "C"
import (
	"errors"
	"unsafe"
)

const (
	HostModeInsert = C.XCB_HOST_MODE_INSERT
	HostModeDelete = C.XCB_HOST_MODE_DELETE

	FamilyInternet          = C.XCB_FAMILY_INTERNET
	FamilyDecnet            = C.XCB_FAMILY_DECNET
	FamilyChaos             = C.XCB_FAMILY_CHAOS
	FamilyServerInterpreted = C.XCB_FAMILY_SERVER_INTERPRETED
	FamilyInternet6         = C.XCB_FAMILY_INTERNET_6
)

func ChangeHosts(mode, family C.uint8_t, address string) error {
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
