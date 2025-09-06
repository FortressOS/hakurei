package system

import (
	"errors"
	"fmt"
	"os"
	"slices"

	"hakurei.app/system/acl"
)

// UpdatePerm calls UpdatePermType with the [Process] criteria.
func (sys *I) UpdatePerm(path string, perms ...acl.Perm) *I {
	sys.UpdatePermType(Process, path, perms...)
	return sys
}

// UpdatePermType maintains [acl.Perms] on a file until its [Enablement] is no longer satisfied.
func (sys *I) UpdatePermType(et Enablement, path string, perms ...acl.Perm) *I {
	sys.ops = append(sys.ops, &aclUpdateOp{et, path, perms})
	return sys
}

// aclUpdateOp implements [I.UpdatePermType].
type aclUpdateOp struct {
	et    Enablement
	path  string
	perms acl.Perms
}

func (a *aclUpdateOp) Type() Enablement { return a.et }

func (a *aclUpdateOp) apply(sys *I) error {
	sys.verbose("applying ACL", a)
	return newOpError("acl", sys.aclUpdate(a.path, sys.uid, a.perms...), false)
}

func (a *aclUpdateOp) revert(sys *I, ec *Criteria) error {
	if ec.hasType(a.Type()) {
		sys.verbose("stripping ACL", a)
		err := sys.aclUpdate(a.path, sys.uid)
		if errors.Is(err, os.ErrNotExist) {
			// the ACL is effectively stripped if the file no longer exists
			sys.verbosef("target of ACL %s no longer exists", a)
			err = nil
		}
		return newOpError("acl", err, true)
	} else {
		sys.verbose("skipping ACL", a)
		return nil
	}
}

func (a *aclUpdateOp) Is(o Op) bool {
	target, ok := o.(*aclUpdateOp)
	return ok && a != nil && target != nil &&
		a.et == target.et &&
		a.path == target.path &&
		slices.Equal(a.perms, target.perms)
}

func (a *aclUpdateOp) Path() string { return a.path }

func (a *aclUpdateOp) String() string {
	return fmt.Sprintf("%s type: %s path: %q",
		a.perms, TypeString(a.et), a.path)
}
