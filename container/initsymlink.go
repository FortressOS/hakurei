package container

import (
	"encoding/gob"
	"fmt"
	"path"
)

func init() { gob.Register(new(SymlinkOp)) }

// Link appends an [Op] that creates a symlink in the container filesystem.
func (f *Ops) Link(target *Absolute, linkName string, dereference bool) *Ops {
	*f = append(*f, &SymlinkOp{target, linkName, dereference})
	return f
}

// SymlinkOp optionally dereferences LinkName and creates a symlink at container path Target.
type SymlinkOp struct {
	Target *Absolute
	// LinkName is an arbitrary uninterpreted pathname.
	LinkName string

	// Dereference causes LinkName to be dereferenced during early.
	Dereference bool
}

func (l *SymlinkOp) Valid() bool { return l != nil && l.Target != nil && l.LinkName != zeroString }

func (l *SymlinkOp) early(_ *setupState, k syscallDispatcher) error {
	if l.Dereference {
		if !isAbs(l.LinkName) {
			return &AbsoluteError{l.LinkName}
		}
		if name, err := k.readlink(l.LinkName); err != nil {
			return err
		} else {
			l.LinkName = name
		}
	}
	return nil
}

func (l *SymlinkOp) apply(state *setupState, k syscallDispatcher) error {
	target := toSysroot(l.Target.String())
	if err := k.mkdirAll(path.Dir(target), state.ParentPerm); err != nil {
		return err
	}
	return k.symlink(l.LinkName, target)
}

func (l *SymlinkOp) Is(op Op) bool {
	vl, ok := op.(*SymlinkOp)
	return ok && l.Valid() && vl.Valid() &&
		l.Target.Is(vl.Target) &&
		l.LinkName == vl.LinkName &&
		l.Dereference == vl.Dereference
}
func (*SymlinkOp) prefix() string { return "creating" }
func (l *SymlinkOp) String() string {
	return fmt.Sprintf("symlink on %q linkname %q", l.Target, l.LinkName)
}
