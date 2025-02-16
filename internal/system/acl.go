package system

import (
	"fmt"
	"slices"

	"git.gensokyo.uk/security/fortify/acl"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
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

func (a *ACL) Type() Enablement {
	return a.et
}

func (a *ACL) apply(sys *I) error {
	fmsg.Verbose("applying ACL", a)
	return fmsg.WrapErrorSuffix(acl.UpdatePerm(a.path, sys.uid, a.perms...),
		fmt.Sprintf("cannot apply ACL entry to %q:", a.path))
}

func (a *ACL) revert(sys *I, ec *Criteria) error {
	if ec.hasType(a) {
		fmsg.Verbose("stripping ACL", a)
		return fmsg.WrapErrorSuffix(acl.UpdatePerm(a.path, sys.uid),
			fmt.Sprintf("cannot strip ACL entry from %q:", a.path))
	} else {
		fmsg.Verbose("skipping ACL", a)
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

func (a *ACL) Path() string {
	return a.path
}

func (a *ACL) String() string {
	return fmt.Sprintf("%s type: %s path: %q",
		a.perms, TypeString(a.et), a.path)
}
