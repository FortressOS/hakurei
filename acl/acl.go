// Package acl implements simple ACL manipulation via libacl.
package acl

/*
#cgo linux pkg-config: --static libacl

#include "acl-update.h"
*/
import "C"

type Perm C.acl_perm_t

const (
	Read    Perm = C.ACL_READ
	Write   Perm = C.ACL_WRITE
	Execute Perm = C.ACL_EXECUTE
)

// Update replaces ACL_USER entry with qualifier uid.
func Update(name string, uid int, perms ...Perm) error {
	var p *Perm
	if len(perms) > 0 {
		p = &perms[0]
	}

	r, err := C.hakurei_acl_update_file_by_uid(
		C.CString(name),
		C.uid_t(uid),
		(*C.acl_perm_t)(p),
		C.size_t(len(perms)),
	)
	if r == 0 {
		return nil
	}
	return err
}
