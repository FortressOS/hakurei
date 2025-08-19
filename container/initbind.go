package container

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	. "syscall"
)

func init() { gob.Register(new(BindMountOp)) }

// Bind appends an [Op] that bind mounts host path [BindMountOp.Source] on container path [BindMountOp.Target].
func (f *Ops) Bind(source, target *Absolute, flags int) *Ops {
	*f = append(*f, &BindMountOp{nil, source, target, flags})
	return f
}

// BindMountOp bind mounts host path Source on container path Target.
// Note that Flags uses bits declared in this package and should not be set with constants in [syscall].
type BindMountOp struct {
	sourceFinal, Source, Target *Absolute

	Flags int
}

const (
	// BindOptional skips nonexistent host paths.
	BindOptional = 1 << iota
	// BindWritable mounts filesystem read-write.
	BindWritable
	// BindDevice allows access to devices (special files) on this filesystem.
	BindDevice
)

func (b *BindMountOp) early(*setupState) error {
	if b.Source == nil || b.Target == nil {
		return EBADE
	}

	if pathname, err := filepath.EvalSymlinks(b.Source.String()); err != nil {
		if os.IsNotExist(err) && b.Flags&BindOptional != 0 {
			// leave sourceFinal as nil
			return nil
		}
		return wrapErrSelf(err)
	} else {
		b.sourceFinal, err = NewAbs(pathname)
		return err
	}
}

func (b *BindMountOp) apply(*setupState) error {
	if b.sourceFinal == nil {
		if b.Flags&BindOptional == 0 {
			// unreachable
			return EBADE
		}
		return nil
	}

	source := toHost(b.sourceFinal.String())
	target := toSysroot(b.Target.String())

	// this perm value emulates bwrap behaviour as it clears bits from 0755 based on
	// op->perms which is never set for any bind setup op so always results in 0700
	if fi, err := os.Stat(source); err != nil {
		return wrapErrSelf(err)
	} else if fi.IsDir() {
		if err = os.MkdirAll(target, 0700); err != nil {
			return wrapErrSelf(err)
		}
	} else if err = ensureFile(target, 0444, 0700); err != nil {
		return err
	}

	var flags uintptr = MS_REC
	if b.Flags&BindWritable == 0 {
		flags |= MS_RDONLY
	}
	if b.Flags&BindDevice == 0 {
		flags |= MS_NODEV
	}

	return hostProc.bindMount(source, target, flags, b.sourceFinal == b.Target)
}

func (b *BindMountOp) Is(op Op) bool {
	vb, ok := op.(*BindMountOp)
	return ok && ((b == nil && vb == nil) || (b != nil && vb != nil &&
		b.Source != nil && vb.Source != nil &&
		b.Source.String() == vb.Source.String() &&
		b.Target != nil && vb.Target != nil &&
		b.Target.String() == vb.Target.String() &&
		b.Flags == vb.Flags))
}
func (*BindMountOp) prefix() string { return "mounting" }
func (b *BindMountOp) String() string {
	if b.Source == nil || b.Target == nil {
		return "<invalid>"
	}
	if b.Source.String() == b.Target.String() {
		return fmt.Sprintf("%q flags %#x", b.Source, b.Flags)
	}
	return fmt.Sprintf("%q on %q flags %#x", b.Source, b.Target, b.Flags)
}
