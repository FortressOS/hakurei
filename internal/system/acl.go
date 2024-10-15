package system

import (
	"fmt"
	"slices"

	"git.ophivana.moe/cat/fortify/acl"
	"git.ophivana.moe/cat/fortify/internal/fmsg"
	"git.ophivana.moe/cat/fortify/internal/state"
	"git.ophivana.moe/cat/fortify/internal/verbose"
)

// UpdatePerm appends an ephemeral acl update Op.
func (sys *I) UpdatePerm(path string, perms ...acl.Perm) {
	sys.UpdatePermType(Process, path, perms...)
}

// UpdatePermType appends an acl update Op.
func (sys *I) UpdatePermType(et state.Enablement, path string, perms ...acl.Perm) {
	sys.lock.Lock()
	defer sys.lock.Unlock()

	sys.ops = append(sys.ops, &ACL{et, path, perms})
}

type ACL struct {
	et    state.Enablement
	path  string
	perms []acl.Perm
}

func (a *ACL) Type() state.Enablement {
	return a.et
}

func (a *ACL) apply(sys *I) error {
	verbose.Println("applying ACL", a, "uid:", sys.uid, "type:", TypeString(a.et), "path:", a.path)
	return fmsg.WrapErrorSuffix(acl.UpdatePerm(a.path, sys.uid, a.perms...),
		fmt.Sprintf("cannot apply ACL entry to %q:", a.path))
}

func (a *ACL) revert(sys *I, ec *Criteria) error {
	if ec.hasType(a) {
		verbose.Println("stripping ACL", a, "uid:", sys.uid, "type:", TypeString(a.et), "path:", a.path)
		return fmsg.WrapErrorSuffix(acl.UpdatePerm(a.path, sys.uid),
			fmt.Sprintf("cannot strip ACL entry from %q:", a.path))
	} else {
		verbose.Println("skipping ACL", a, "uid:", sys.uid, "tag:", TypeString(a.et), "path:", a.path)
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
	var s = []byte("---")
	for _, p := range a.perms {
		switch p {
		case acl.Read:
			s[0] = 'r'
		case acl.Write:
			s[1] = 'w'
		case acl.Execute:
			s[2] = 'x'
		}
	}
	return string(s)
}
