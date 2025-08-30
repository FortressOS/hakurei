package system

import (
	"hakurei.app/system/internal/xcb"
)

// ChangeHosts appends an X11 ChangeHosts command Op.
func (sys *I) ChangeHosts(username string) *I {
	sys.lock.Lock()
	defer sys.lock.Unlock()

	sys.ops = append(sys.ops, XHost(username))

	return sys
}

type XHost string

func (x XHost) Type() Enablement { return EX11 }

func (x XHost) apply(*I) error {
	msg.Verbosef("inserting entry %s to X11", x)
	return newOpError("xhost",
		xcb.ChangeHosts(xcb.HostModeInsert, xcb.FamilyServerInterpreted, "localuser\x00"+string(x)), false)
}

func (x XHost) revert(_ *I, ec *Criteria) error {
	if ec.hasType(x) {
		msg.Verbosef("deleting entry %s from X11", x)
		return newOpError("xhost",
			xcb.ChangeHosts(xcb.HostModeDelete, xcb.FamilyServerInterpreted, "localuser\x00"+string(x)), false)
	} else {
		msg.Verbosef("skipping entry %s in X11", x)
		return nil
	}
}

func (x XHost) Is(o Op) bool   { x0, ok := o.(XHost); return ok && x == x0 }
func (x XHost) Path() string   { return string(x) }
func (x XHost) String() string { return string("SI:localuser:" + x) }
