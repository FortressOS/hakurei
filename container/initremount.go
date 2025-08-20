package container

import (
	"encoding/gob"
	"fmt"
	"syscall"
)

func init() { gob.Register(new(RemountOp)) }

// Remount appends an [Op] that applies [RemountOp.Flags] on container path [RemountOp.Target].
func (f *Ops) Remount(target *Absolute, flags uintptr) *Ops {
	*f = append(*f, &RemountOp{target, flags})
	return f
}

// RemountOp remounts Target with Flags.
type RemountOp struct {
	Target *Absolute
	Flags  uintptr
}

func (*RemountOp) early(*setupState) error { return nil }
func (r *RemountOp) apply(*setupState) error {
	if r.Target == nil {
		return syscall.EBADE
	}
	return wrapErrSuffix(hostProc.remount(toSysroot(r.Target.String()), r.Flags),
		fmt.Sprintf("cannot remount %q:", r.Target))
}

func (r *RemountOp) Is(op Op) bool {
	vr, ok := op.(*RemountOp)
	return ok && ((r == nil && vr == nil) ||
		(r.Target != nil && vr.Target != nil && r.Target.Is(vr.Target)) &&
			r.Flags == vr.Flags)
}
func (*RemountOp) prefix() string   { return "remounting" }
func (r *RemountOp) String() string { return fmt.Sprintf("%q flags %#x", r.Target, r.Flags) }
