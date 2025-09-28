package system

import (
	"fmt"

	"hakurei.app/hst"
)

// Link calls LinkFileType with the [Process] criteria.
func (sys *I) Link(oldname, newname string) *I { return sys.LinkFileType(Process, oldname, newname) }

// LinkFileType maintains a hardlink until its [Enablement] is no longer satisfied.
func (sys *I) LinkFileType(et hst.Enablement, oldname, newname string) *I {
	sys.ops = append(sys.ops, &hardlinkOp{et, newname, oldname})
	return sys
}

// hardlinkOp implements [I.LinkFileType].
type hardlinkOp struct {
	et       hst.Enablement
	dst, src string
}

func (l *hardlinkOp) Type() hst.Enablement { return l.et }

func (l *hardlinkOp) apply(sys *I) error {
	sys.msg.Verbose("linking", l)
	return newOpError("hardlink", sys.link(l.src, l.dst), false)
}

func (l *hardlinkOp) revert(sys *I, ec *Criteria) error {
	if ec.hasType(l.Type()) {
		sys.msg.Verbosef("removing hard link %q", l.dst)
		return newOpError("hardlink", sys.remove(l.dst), true)
	} else {
		sys.msg.Verbosef("skipping hard link %q", l.dst)
		return nil
	}
}

func (l *hardlinkOp) Is(o Op) bool {
	target, ok := o.(*hardlinkOp)
	return ok && l != nil && target != nil && *l == *target
}

func (l *hardlinkOp) Path() string   { return l.src }
func (l *hardlinkOp) String() string { return fmt.Sprintf("%q from %q", l.dst, l.src) }
