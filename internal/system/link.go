package system

import (
	"fmt"
	"os"
)

// Link registers an Op that links dst to src.
func (sys *I) Link(oldname, newname string) *I { return sys.LinkFileType(Process, oldname, newname) }

// LinkFileType registers a file linking Op labelled with type et.
func (sys *I) LinkFileType(et Enablement, oldname, newname string) *I {
	sys.lock.Lock()
	defer sys.lock.Unlock()

	sys.ops = append(sys.ops, &Hardlink{et, newname, oldname})

	return sys
}

type Hardlink struct {
	et       Enablement
	dst, src string
}

func (l *Hardlink) Type() Enablement { return l.et }

func (l *Hardlink) apply(sys *I) error {
	sys.println("linking", l)
	return sys.wrapErrSuffix(os.Link(l.src, l.dst),
		fmt.Sprintf("cannot link %q:", l.dst))
}

func (l *Hardlink) revert(sys *I, ec *Criteria) error {
	if ec.hasType(l) {
		sys.printf("removing hard link %q", l.dst)
		return sys.wrapErrSuffix(os.Remove(l.dst),
			fmt.Sprintf("cannot remove hard link %q:", l.dst))
	} else {
		sys.printf("skipping hard link %q", l.dst)
		return nil
	}
}

func (l *Hardlink) Is(o Op) bool   { l0, ok := o.(*Hardlink); return ok && l0 != nil && *l == *l0 }
func (l *Hardlink) Path() string   { return l.src }
func (l *Hardlink) String() string { return fmt.Sprintf("%q from %q", l.dst, l.src) }
