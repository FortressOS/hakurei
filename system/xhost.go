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

func (x xhostOp) apply(sys *I) error {
	sys.verbosef("inserting entry %s to X11", x)
	return newOpError("xhost",
		sys.xcbChangeHosts(xcb.HostModeInsert, xcb.FamilyServerInterpreted, "localuser\x00"+string(x)), false)
}

func (x xhostOp) revert(sys *I, ec *Criteria) error {
	if ec.hasType(x.Type()) {
		sys.verbosef("deleting entry %s from X11", x)
		return newOpError("xhost",
			sys.xcbChangeHosts(xcb.HostModeDelete, xcb.FamilyServerInterpreted, "localuser\x00"+string(x)), true)
	} else {
		sys.verbosef("skipping entry %s in X11", x)
		return nil
	}
}

func (x xhostOp) Is(o Op) bool   { target, ok := o.(xhostOp); return ok && x == target }
func (x xhostOp) Path() string   { return "/tmp/.X11-unix" }
func (x xhostOp) String() string { return string("SI:localuser:" + x) }
