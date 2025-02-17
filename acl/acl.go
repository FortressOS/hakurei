// Package acl implements simple ACL manipulation via libacl.
package acl

import (
	"errors"
	"runtime"
	"syscall"
	"unsafe"
)

/*
#cgo linux pkg-config: --static libacl

#include <stdlib.h>
#include <sys/acl.h>
#include <acl/libacl.h>

static acl_t _go_acl_get_file(const char *path_p, acl_type_t type) {
	acl_t acl = acl_get_file(path_p, type);
	free((void *)path_p);
	return acl;
}

static int _go_acl_set_file(const char *path_p, acl_type_t type, acl_t acl) {
	if (acl_valid(acl) != 0) {
		return -1;
	}

	int ret = acl_set_file(path_p, type, acl);
	free((void *)path_p);
	return ret;
}
*/
import "C"

func getFile(name string, t C.acl_type_t) (*ACL, error) {
	a, err := C._go_acl_get_file(C.CString(name), t)
	if errors.Is(err, syscall.ENODATA) {
		err = nil
	}

	return newACL(a), err
}

func (acl *ACL) setFile(name string, t C.acl_type_t) error {
	_, err := C._go_acl_set_file(C.CString(name), t, acl.acl)
	return err
}

func newACL(a C.acl_t) *ACL {
	acl := &ACL{a}
	runtime.SetFinalizer(acl, (*ACL).free)
	return acl
}

type ACL struct {
	acl C.acl_t
}

func (acl *ACL) free() {
	C.acl_free(unsafe.Pointer(acl.acl))

	// no need for a finalizer anymore
	runtime.SetFinalizer(acl, nil)
}

const (
	Read    = C.ACL_READ
	Write   = C.ACL_WRITE
	Execute = C.ACL_EXECUTE

	TypeDefault = C.ACL_TYPE_DEFAULT
	TypeAccess  = C.ACL_TYPE_ACCESS

	UndefinedTag = C.ACL_UNDEFINED_TAG
	UserObj      = C.ACL_USER_OBJ
	User         = C.ACL_USER
	GroupObj     = C.ACL_GROUP_OBJ
	Group        = C.ACL_GROUP
	Mask         = C.ACL_MASK
	Other        = C.ACL_OTHER
)

type (
	Perm C.acl_perm_t
)

func (acl *ACL) removeEntry(tt C.acl_tag_t, tq int) error {
	var e C.acl_entry_t

	// get first entry
	if r, err := C.acl_get_entry(acl.acl, C.ACL_FIRST_ENTRY, &e); err != nil {
		return err
	} else if r == 0 {
		// return on acl with no entries
		return nil
	}

	for {
		if r, err := C.acl_get_entry(acl.acl, C.ACL_NEXT_ENTRY, &e); err != nil {
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
			_, err := C.acl_delete_entry(acl.acl, e)
			return err
		}
	}
}

// Update replaces ACL_USER entry with qualifier uid.
func Update(name string, uid int, perms ...Perm) error {
	// read acl from file
	a, err := getFile(name, TypeAccess)
	if err != nil {
		return err
	}
	// free acl on return if get is successful
	defer a.free()

	// remove existing entry
	if err = a.removeEntry(User, uid); err != nil {
		return err
	}

	// create new entry if perms are passed
	if len(perms) > 0 {
		// create new acl entry
		var e C.acl_entry_t
		if _, err = C.acl_create_entry(&a.acl, &e); err != nil {
			return err
		}

		// get perm set of new entry
		var p C.acl_permset_t
		if _, err = C.acl_get_permset(e, &p); err != nil {
			return err
		}

		// add target perms
		for _, perm := range perms {
			if _, err = C.acl_add_perm(p, C.acl_perm_t(perm)); err != nil {
				return err
			}
		}

		// set perm set to new entry
		if _, err = C.acl_set_permset(e, p); err != nil {
			return err
		}

		// set user tag to new entry
		if _, err = C.acl_set_tag_type(e, User); err != nil {
			return err
		}

		// set qualifier (uid) to new entry
		if _, err = C.acl_set_qualifier(e, unsafe.Pointer(&uid)); err != nil {
			return err
		}
	}

	// calculate mask after update
	if _, err = C.acl_calc_mask(&a.acl); err != nil {
		return err
	}

	// write acl to file
	return a.setFile(name, TypeAccess)
}
