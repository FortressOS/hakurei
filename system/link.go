package system

import (
	"fmt"
	"os"
)

// Link appends [HardlinkOp] to [I] the [Process] criteria.
func (sys *I) Link(oldname, newname string) *I { return sys.LinkFileType(Process, oldname, newname) }

// LinkFileType appends [HardlinkOp] to [I].
func (sys *I) LinkFileType(et Enablement, oldname, newname string) *I {
	sys.ops = append(sys.ops, &HardlinkOp{et, newname, oldname})
	return sys
}

// HardlinkOp maintains a hardlink until its [Enablement] is no longer satisfied.
type HardlinkOp struct {
	et       Enablement
	dst, src string
}

func (l *HardlinkOp) Type() Enablement { return l.et }

func (l *HardlinkOp) apply(*I) error {
	msg.Verbose("linking", l)
	return newOpError("hardlink", os.Link(l.src, l.dst), false)
}

func (l *HardlinkOp) revert(_ *I, ec *Criteria) error {
	if ec.hasType(l.Type()) {
		msg.Verbosef("removing hard link %q", l.dst)
		return newOpError("hardlink", os.Remove(l.dst), true)
	} else {
		msg.Verbosef("skipping hard link %q", l.dst)
		return nil
	}
}

func (l *HardlinkOp) Is(o Op) bool {
	target, ok := o.(*HardlinkOp)
	return ok && l != nil && target != nil && *l == *target
}

func (l *HardlinkOp) Path() string   { return l.src }
func (l *HardlinkOp) String() string { return fmt.Sprintf("%q from %q", l.dst, l.src) }
