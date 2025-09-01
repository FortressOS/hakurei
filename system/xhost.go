package system

import (
	"hakurei.app/system/internal/xcb"
)

// ChangeHosts appends [XHostOp] to [I].
func (sys *I) ChangeHosts(username string) *I {
	sys.ops = append(sys.ops, XHostOp(username))
	return sys
}

// XHostOp inserts the target user into X11 hosts and deletes it once its [Enablement] is no longer satisfied.
type XHostOp string

func (x XHostOp) Type() Enablement { return EX11 }

func (x XHostOp) apply(*I) error {
	msg.Verbosef("inserting entry %s to X11", x)
	return newOpError("xhost",
		xcb.ChangeHosts(xcb.HostModeInsert, xcb.FamilyServerInterpreted, "localuser\x00"+string(x)), false)
}

func (x XHostOp) revert(_ *I, ec *Criteria) error {
	if ec.hasType(x) {
		msg.Verbosef("deleting entry %s from X11", x)
		return newOpError("xhost",
			xcb.ChangeHosts(xcb.HostModeDelete, xcb.FamilyServerInterpreted, "localuser\x00"+string(x)), false)
	} else {
		msg.Verbosef("skipping entry %s in X11", x)
		return nil
	}
}

func (x XHostOp) Is(o Op) bool   { target, ok := o.(XHostOp); return ok && x == target }
func (x XHostOp) Path() string   { return string(x) }
func (x XHostOp) String() string { return string("SI:localuser:" + x) }
