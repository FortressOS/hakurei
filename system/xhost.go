package system

import (
	"fmt"

	"git.gensokyo.uk/security/fortify/system/internal/xcb"
)

// ChangeHosts appends an X11 ChangeHosts command Op.
func (sys *I) ChangeHosts(username string) *I {
	sys.lock.Lock()
	defer sys.lock.Unlock()

	sys.ops = append(sys.ops, XHost(username))

	return sys
}

type XHost string

func (x XHost) Type() Enablement {
	return EX11
}

func (x XHost) apply(sys *I) error {
	sys.printf("inserting entry %s to X11", x)
	return sys.wrapErrSuffix(xcb.ChangeHosts(xcb.HostModeInsert, xcb.FamilyServerInterpreted, "localuser\x00"+string(x)),
		fmt.Sprintf("cannot insert entry %s to X11:", x))
}

func (x XHost) revert(sys *I, ec *Criteria) error {
	if ec.hasType(x) {
		sys.printf("deleting entry %s from X11", x)
		return sys.wrapErrSuffix(xcb.ChangeHosts(xcb.HostModeDelete, xcb.FamilyServerInterpreted, "localuser\x00"+string(x)),
			fmt.Sprintf("cannot delete entry %s from X11:", x))
	} else {
		sys.printf("skipping entry %s in X11", x)
		return nil
	}
}

func (x XHost) Is(o Op) bool {
	x0, ok := o.(XHost)
	return ok && x == x0
}

func (x XHost) Path() string {
	return string(x)
}

func (x XHost) String() string {
	return string("SI:localuser:" + x)
}
