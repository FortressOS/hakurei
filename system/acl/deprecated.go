// Package acl exposes the internal/acl package.
//
// Deprecated: This package will be removed in 0.4.
package acl

import (
	_ "unsafe" // for go:linkname

	"hakurei.app/internal/acl"
)

type Perm = acl.Perm

const (
	Read    = acl.Read
	Write   = acl.Write
	Execute = acl.Execute
)

// Update replaces ACL_USER entry with qualifier uid.
//
//go:linkname Update hakurei.app/internal/acl.Update
func Update(name string, uid int, perms ...Perm) error

type Perms = acl.Perms
