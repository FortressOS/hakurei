package acl

import "unsafe"

//#include <stdlib.h>
//#include <sys/acl.h>
//#include <acl/libacl.h>
//#cgo linux LDFLAGS: -lacl
import "C"

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
	Perm  C.acl_perm_t
	Perms []Perm
)

func (ps Perms) String() string {
	var s = []byte("---")
	for _, p := range ps {
		switch p {
		case Read:
			s[0] = 'r'
		case Write:
			s[1] = 'w'
		case Execute:
			s[2] = 'x'
		}
	}
	return string(s)
}

func UpdatePerm(path string, uid int, perms ...Perm) error {
	// read acl from file
	a, err := aclGetFile(path, TypeAccess)
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
	if _, err = C.acl_calc_mask(&a.val); err != nil {
		return err
	}

	// write acl to file
	return a.setFile(path, TypeAccess)
}
