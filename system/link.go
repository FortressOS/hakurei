package system

import (
	"fmt"
	"os"
)

// Link calls LinkFileType with the [Process] criteria.
func (sys *I) Link(oldname, newname string) *I { return sys.LinkFileType(Process, oldname, newname) }

// LinkFileType maintains a hardlink until its [Enablement] is no longer satisfied.
func (sys *I) LinkFileType(et Enablement, oldname, newname string) *I {
	sys.ops = append(sys.ops, &hardlinkOp{et, newname, oldname})
	return sys
}

// hardlinkOp implements [I.LinkFileType].
type hardlinkOp struct {
	et       Enablement
	dst, src string
}

func (l *hardlinkOp) Type() Enablement { return l.et }

func (l *hardlinkOp) apply(*I) error {
	msg.Verbose("linking", l)
	return newOpError("hardlink", os.Link(l.src, l.dst), false)
}

func (l *hardlinkOp) revert(_ *I, ec *Criteria) error {
	if ec.hasType(l.Type()) {
		msg.Verbosef("removing hard link %q", l.dst)
		return newOpError("hardlink", os.Remove(l.dst), true)
	} else {
		msg.Verbosef("skipping hard link %q", l.dst)
		return nil
	}
}

func (l *hardlinkOp) Is(o Op) bool {
	target, ok := o.(*hardlinkOp)
	return ok && l != nil && target != nil && *l == *target
}

func (l *hardlinkOp) Path() string   { return l.src }
func (l *hardlinkOp) String() string { return fmt.Sprintf("%q from %q", l.dst, l.src) }
