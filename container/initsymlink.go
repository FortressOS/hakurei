package container

import (
	"encoding/gob"
	"fmt"
	"os"
	"path"
	"syscall"
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

func (l *SymlinkOp) early(*setupState) error {
	if l.Dereference {
		if !isAbs(l.LinkName) {
			return msg.WrapErr(syscall.EBADE, fmt.Sprintf("path %q is not absolute", l.LinkName))
		}
		if name, err := os.Readlink(l.LinkName); err != nil {
			return wrapErrSelf(err)
		} else {
			l.LinkName = name
		}
	}
	return nil
}

func (l *SymlinkOp) apply(state *setupState) error {
	if l.Target == nil {
		return syscall.EBADE
	}
	target := toSysroot(l.Target.String())
	if err := os.MkdirAll(path.Dir(target), state.ParentPerm); err != nil {
		return wrapErrSelf(err)
	}
	if err := os.Symlink(l.LinkName, target); err != nil {
		return wrapErrSelf(err)
	}
	return nil
}

func (l *SymlinkOp) Is(op Op) bool { vl, ok := op.(*SymlinkOp); return ok && *l == *vl }
func (*SymlinkOp) prefix() string  { return "creating" }
func (l *SymlinkOp) String() string {
	return fmt.Sprintf("symlink on %q linkname %q", l.Target, l.LinkName)
}
