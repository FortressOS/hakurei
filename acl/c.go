package acl

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"
)

//#include <stdlib.h>
//#include <sys/acl.h>
//#include <acl/libacl.h>
//#cgo linux LDFLAGS: -lacl
import "C"

type acl struct {
	val   C.acl_t
	freed bool
}

func aclGetFile(path string, t C.acl_type_t) (*acl, error) {
	p := C.CString(path)
	a, err := C.acl_get_file(p, t)
	C.free(unsafe.Pointer(p))

	if errors.Is(err, syscall.ENODATA) {
		err = nil
	}
	return &acl{val: a, freed: false}, err
}

func (a *acl) setFile(path string, t C.acl_type_t) error {
	if C.acl_valid(a.val) != 0 {
		return fmt.Errorf("invalid acl")
	}

	p := C.CString(path)
	_, err := C.acl_set_file(p, t, a.val)
	C.free(unsafe.Pointer(p))
	return err
}

func (a *acl) removeEntry(tt C.acl_tag_t, tq int) error {
	var e C.acl_entry_t

	// get first entry
	if r, err := C.acl_get_entry(a.val, C.ACL_FIRST_ENTRY, &e); err != nil {
		return err
	} else if r == 0 {
		// return on acl with no entries
		return nil
	}

	for {
		if r, err := C.acl_get_entry(a.val, C.ACL_NEXT_ENTRY, &e); err != nil {
			return err
		} else if r == 0 {
			// return on drained acl
			return nil
		}

		var (
			q int
			t C.acl_tag_t
		)

		// get current entry tag type
		if _, err := C.acl_get_tag_type(e, &t); err != nil {
			return err
		}

		// get current entry qualifier
		if rq, err := C.acl_get_qualifier(e); err != nil {
			// neither ACL_USER nor ACL_GROUP
			if errors.Is(err, syscall.EINVAL) {
				continue
			}

			return err
		} else {
			q = *(*int)(rq)
			C.acl_free(rq)
		}

		// delete on match
		if t == tt && q == tq {
			_, err := C.acl_delete_entry(a.val, e)
			return err
		}
	}
}

func (a *acl) free() {
	if a.freed {
		panic("acl already freed")
	}
	C.acl_free(unsafe.Pointer(a.val))
	a.freed = true
}
