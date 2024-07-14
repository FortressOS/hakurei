package main

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

const (
	aclRead    = C.ACL_READ
	aclWrite   = C.ACL_WRITE
	aclExecute = C.ACL_EXECUTE

	aclTypeDefault = C.ACL_TYPE_DEFAULT
	aclTypeAccess  = C.ACL_TYPE_ACCESS

	aclUndefinedTag = C.ACL_UNDEFINED_TAG
	aclUserObj      = C.ACL_USER_OBJ
	aclUser         = C.ACL_USER
	aclGroupObj     = C.ACL_GROUP_OBJ
	aclGroup        = C.ACL_GROUP
	aclMask         = C.ACL_MASK
	aclOther        = C.ACL_OTHER
)

type acl struct {
	val   C.acl_t
	freed bool
}

func aclUpdatePerm(path string, uid int, perms ...C.acl_perm_t) error {
	// read acl from file
	a, err := aclGetFile(path, aclTypeAccess)
	if err != nil {
		return err
	}
	// free acl on return if get is successful
	defer a.free()

	// remove existing entry
	if err = a.removeEntry(aclUser, uid); err != nil {
		return err
	}

	// create new entry if perms are passed
	if len(perms) > 0 {
		// create new acl entry
		var e C.acl_entry_t
		if _, err = C.acl_create_entry(&a.val, &e); err != nil {
			return err
		}

		// get perm set of new entry
		var p C.acl_permset_t
		if _, err = C.acl_get_permset(e, &p); err != nil {
			return err
		}

		// add target perms
		for _, perm := range perms {
			if _, err = C.acl_add_perm(p, perm); err != nil {
				return err
			}
		}

		// set perm set to new entry
		if _, err = C.acl_set_permset(e, p); err != nil {
			return err
		}

		// set user tag to new entry
		if _, err = C.acl_set_tag_type(e, aclUser); err != nil {
			return err
		}

		// set qualifier (uid) to new entry
		if _, err = C.acl_set_qualifier(e, unsafe.Pointer(&uid)); err != nil {
			return err
		}
	}

	// calculate mask after update
	if _, err = C.acl_calc_mask(&a.val); err != nil {
		return err
	}

	// write acl to file
	return a.setFile(path, aclTypeAccess)
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
