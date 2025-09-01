package system

import (
	"errors"
	"fmt"
	"os"
	"slices"

	"hakurei.app/system/acl"
)

// UpdatePerm appends [ACLUpdateOp] to [I] with the [Process] criteria.
func (sys *I) UpdatePerm(path string, perms ...acl.Perm) *I {
	sys.UpdatePermType(Process, path, perms...)
	return sys
}

// UpdatePermType appends [ACLUpdateOp] to [I].
func (sys *I) UpdatePermType(et Enablement, path string, perms ...acl.Perm) *I {
	sys.ops = append(sys.ops, &ACLUpdateOp{et, path, perms})
	return sys
}

// ACLUpdateOp maintains [acl.Perms] on a file until its [Enablement] is no longer satisfied.
type ACLUpdateOp struct {
	et    Enablement
	path  string
	perms acl.Perms
}

func (a *ACLUpdateOp) Type() Enablement { return a.et }

func (a *ACLUpdateOp) apply(sys *I) error {
	msg.Verbose("applying ACL", a)
	return newOpError("acl", acl.Update(a.path, sys.uid, a.perms...), false)
}

func (a *ACLUpdateOp) revert(sys *I, ec *Criteria) error {
	if ec.hasType(a) {
		msg.Verbose("stripping ACL", a)
		err := acl.Update(a.path, sys.uid)
		if errors.Is(err, os.ErrNotExist) {
			// the ACL is effectively stripped if the file no longer exists
			msg.Verbosef("target of ACL %s no longer exists", a)
			err = nil
		}
		return newOpError("acl", err, true)
	} else {
		msg.Verbose("skipping ACL", a)
		return nil
	}
}

func (a *ACLUpdateOp) Is(o Op) bool {
	target, ok := o.(*ACLUpdateOp)
	return ok && a != nil && target != nil &&
		a.et == target.et &&
		a.path == target.path &&
		slices.Equal(a.perms, target.perms)
}

func (a *ACLUpdateOp) Path() string { return a.path }

func (a *ACLUpdateOp) String() string {
	return fmt.Sprintf("%s type: %s path: %q",
		a.perms, TypeString(a.et), a.path)
}
