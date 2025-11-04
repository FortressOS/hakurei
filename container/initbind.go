package container

import (
	"encoding/gob"
	"fmt"
	"os"
	"syscall"

	"hakurei.app/container/check"
	"hakurei.app/container/std"
)

func init() { gob.Register(new(BindMountOp)) }

// Bind appends an [Op] that bind mounts host path [BindMountOp.Source] on container path [BindMountOp.Target].
func (f *Ops) Bind(source, target *check.Absolute, flags int) *Ops {
	*f = append(*f, &BindMountOp{nil, source, target, flags})
	return f
}

// BindMountOp bind mounts host path Source on container path Target.
// Note that Flags uses bits declared in this package and should not be set with constants in [syscall].
type BindMountOp struct {
	sourceFinal, Source, Target *check.Absolute

	Flags int
}

func (b *BindMountOp) Valid() bool {
	return b != nil &&
		b.Source != nil && b.Target != nil &&
		b.Flags&(std.BindOptional|std.BindEnsure) != (std.BindOptional|std.BindEnsure)
}

func (b *BindMountOp) early(_ *setupState, k syscallDispatcher) error {
	if b.Flags&std.BindEnsure != 0 {
		if err := k.mkdirAll(b.Source.String(), 0700); err != nil {
			return err
		}
	}

	if pathname, err := k.evalSymlinks(b.Source.String()); err != nil {
		if os.IsNotExist(err) && b.Flags&std.BindOptional != 0 {
			// leave sourceFinal as nil
			return nil
		}
		return err
	} else {
		b.sourceFinal, err = check.NewAbs(pathname)
		return err
	}
}

func (b *BindMountOp) apply(state *setupState, k syscallDispatcher) error {
	if b.sourceFinal == nil {
		if b.Flags&std.BindOptional == 0 {
			// unreachable
			return OpStateError("bind")
		}
		return nil
	}

	source := toHost(b.sourceFinal.String())
	target := toSysroot(b.Target.String())

	// this perm value emulates bwrap behaviour as it clears bits from 0755 based on
	// op->perms which is never set for any bind setup op so always results in 0700
	if fi, err := k.stat(source); err != nil {
		return err
	} else if fi.IsDir() {
		if err = k.mkdirAll(target, 0700); err != nil {
			return err
		}
	} else if err = k.ensureFile(target, 0444, 0700); err != nil {
		return err
	}

	var flags uintptr = syscall.MS_REC
	if b.Flags&std.BindWritable == 0 {
		flags |= syscall.MS_RDONLY
	}
	if b.Flags&std.BindDevice == 0 {
		flags |= syscall.MS_NODEV
	}

	if b.sourceFinal.String() == b.Target.String() {
		state.Verbosef("mounting %q flags %#x", target, flags)
	} else {
		state.Verbosef("mounting %q on %q flags %#x", source, target, flags)
	}
	return k.bindMount(state, source, target, flags)
}

func (b *BindMountOp) Is(op Op) bool {
	vb, ok := op.(*BindMountOp)
	return ok && b.Valid() && vb.Valid() &&
		b.Source.Is(vb.Source) &&
		b.Target.Is(vb.Target) &&
		b.Flags == vb.Flags
}
func (*BindMountOp) prefix() (string, bool) { return "mounting", false }
func (b *BindMountOp) String() string {
	if b.Source == nil || b.Target == nil {
		return "<invalid>"
	}
	if b.Source.String() == b.Target.String() {
		return fmt.Sprintf("%q flags %#x", b.Source, b.Flags)
	}
	return fmt.Sprintf("%q on %q flags %#x", b.Source, b.Target, b.Flags)
}
