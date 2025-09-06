package system

import (
	"hakurei.app/system/internal/xcb"
)

// ChangeHosts inserts the target user into X11 hosts and deletes it once its [Enablement] is no longer satisfied.
func (sys *I) ChangeHosts(username string) *I {
	sys.ops = append(sys.ops, xhostOp(username))
	return sys
}

// xhostOp implements [I.ChangeHosts].
type xhostOp string

func (x xhostOp) Type() Enablement { return EX11 }

func (x xhostOp) apply(*I) error {
	msg.Verbosef("inserting entry %s to X11", x)
	return newOpError("xhost",
		xcb.ChangeHosts(xcb.HostModeInsert, xcb.FamilyServerInterpreted, "localuser\x00"+string(x)), false)
}

func (x xhostOp) revert(_ *I, ec *Criteria) error {
	if ec.hasType(x.Type()) {
		msg.Verbosef("deleting entry %s from X11", x)
		return newOpError("xhost",
			xcb.ChangeHosts(xcb.HostModeDelete, xcb.FamilyServerInterpreted, "localuser\x00"+string(x)), false)
	} else {
		msg.Verbosef("skipping entry %s in X11", x)
		return nil
	}
}

func (x xhostOp) Is(o Op) bool   { target, ok := o.(xhostOp); return ok && x == target }
func (x xhostOp) Path() string   { return string(x) }
func (x xhostOp) String() string { return string("SI:localuser:" + x) }
