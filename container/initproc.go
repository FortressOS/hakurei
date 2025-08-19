package container

import (
	"encoding/gob"
	"fmt"
	"os"
	. "syscall"
)

func init() { gob.Register(new(MountProcOp)) }

// Proc appends an [Op] that mounts a private instance of proc.
func (f *Ops) Proc(target *Absolute) *Ops {
	*f = append(*f, &MountProcOp{target})
	return f
}

// MountProcOp mounts a new instance of [FstypeProc] on container path Target.
type MountProcOp struct {
	Target *Absolute
}

func (p *MountProcOp) early(*setupState) error { return nil }
func (p *MountProcOp) apply(state *setupState) error {
	if p.Target == nil {
		return EBADE
	}
	target := toSysroot(p.Target.String())
	if err := os.MkdirAll(target, state.ParentPerm); err != nil {
		return wrapErrSelf(err)
	}
	return wrapErrSuffix(Mount(SourceProc, target, FstypeProc, MS_NOSUID|MS_NOEXEC|MS_NODEV, zeroString),
		fmt.Sprintf("cannot mount proc on %q:", p.Target.String()))
}

func (p *MountProcOp) Is(op Op) bool {
	vp, ok := op.(*MountProcOp)
	return ok && ((p == nil && vp == nil) || p == vp)
}
func (*MountProcOp) prefix() string   { return "mounting" }
func (p *MountProcOp) String() string { return fmt.Sprintf("proc on %q", p.Target) }
