package system

import (
	"errors"
	"fmt"
	"os"
	"slices"

	"git.gensokyo.uk/security/hakurei/system/acl"
)

// UpdatePerm appends an ephemeral acl update Op.
func (sys *I) UpdatePerm(path string, perms ...acl.Perm) *I {
	sys.UpdatePermType(Process, path, perms...)

	return sys
}

// UpdatePermType appends an acl update Op.
func (sys *I) UpdatePermType(et Enablement, path string, perms ...acl.Perm) *I {
	sys.lock.Lock()
	defer sys.lock.Unlock()

	sys.ops = append(sys.ops, &ACL{et, path, perms})

	return sys
}

type ACL struct {
	et    Enablement
	path  string
	perms acl.Perms
}

func (a *ACL) Type() Enablement { return a.et }

func (a *ACL) apply(sys *I) error {
	msg.Verbose("applying ACL", a)
	return wrapErrSuffix(acl.Update(a.path, sys.uid, a.perms...),
		fmt.Sprintf("cannot apply ACL entry to %q:", a.path))
}

func (a *ACL) revert(sys *I, ec *Criteria) error {
	if ec.hasType(a) {
		msg.Verbose("stripping ACL", a)
		err := acl.Update(a.path, sys.uid)
		if errors.Is(err, os.ErrNotExist) {
			// the ACL is effectively stripped if the file no longer exists
			msg.Verbosef("target of ACL %s no longer exists", a)
			err = nil
		}
		return wrapErrSuffix(err,
			fmt.Sprintf("cannot strip ACL entry from %q:", a.path))
	} else {
		msg.Verbose("skipping ACL", a)
		return nil
	}
}

func (a *ACL) Is(o Op) bool {
	a0, ok := o.(*ACL)
	return ok && a0 != nil &&
		a.et == a0.et &&
		a.path == a0.path &&
		slices.Equal(a.perms, a0.perms)
}

func (a *ACL) Path() string { return a.path }

func (a *ACL) String() string {
	return fmt.Sprintf("%s type: %s path: %q",
		a.perms, TypeString(a.et), a.path)
}
