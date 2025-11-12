// Package acl exposes the internal/system/acl package.
//
// Deprecated: This package will be removed in 0.4.
package acl

import (
	_ "unsafe" // for go:linkname

	"hakurei.app/internal/system/acl"
)

type Perm = acl.Perm

const (
	Read    = acl.Read
	Write   = acl.Write
	Execute = acl.Execute
)

// Update replaces ACL_USER entry with qualifier uid.
//
//go:linkname Update hakurei.app/internal/system/acl.Update
func Update(name string, uid int, perms ...Perm) error

type Perms = acl.Perms
