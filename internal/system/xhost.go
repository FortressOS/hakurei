package system

import (
	"fmt"

	"git.ophivana.moe/cat/fortify/internal/fmsg"
	"git.ophivana.moe/cat/fortify/internal/verbose"
	"git.ophivana.moe/cat/fortify/xcb"
)

// ChangeHosts appends an X11 ChangeHosts command Op.
func (sys *I) ChangeHosts(username string) {
	sys.lock.Lock()
	defer sys.lock.Unlock()

	sys.ops = append(sys.ops, XHost(username))
}

type XHost string

func (x XHost) Type() Enablement {
	return EX11
}

func (x XHost) apply(_ *I) error {
	verbose.Printf("inserting entry %s to X11\n", x)
	return fmsg.WrapErrorSuffix(xcb.ChangeHosts(xcb.HostModeInsert, xcb.FamilyServerInterpreted, "localuser\x00"+string(x)),
		fmt.Sprintf("cannot insert entry %s to X11:", x))
}

func (x XHost) revert(_ *I, ec *Criteria) error {
	if ec.hasType(x) {
		verbose.Printf("deleting entry %s from X11\n", x)
		return fmsg.WrapErrorSuffix(xcb.ChangeHosts(xcb.HostModeDelete, xcb.FamilyServerInterpreted, "localuser\x00"+string(x)),
			fmt.Sprintf("cannot delete entry %s from X11:", x))
	} else {
		verbose.Printf("skipping entry %s in X11\n", x)
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
