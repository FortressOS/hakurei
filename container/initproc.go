package container

import (
	"encoding/gob"
	"fmt"
	. "syscall"
)

func init() { gob.Register(new(MountProcOp)) }

// Proc appends an [Op] that mounts a private instance of proc.
func (f *Ops) Proc(target *Absolute) *Ops {
	*f = append(*f, &MountProcOp{target})
	return f
}

// MountProcOp mounts a new instance of [FstypeProc] on container path Target.
type MountProcOp struct{ Target *Absolute }

func (p *MountProcOp) Valid() bool                                { return p != nil && p.Target != nil }
func (p *MountProcOp) early(*setupState, syscallDispatcher) error { return nil }
func (p *MountProcOp) apply(state *setupState, k syscallDispatcher) error {
	target := toSysroot(p.Target.String())
	if err := k.mkdirAll(target, state.ParentPerm); err != nil {
		return err
	}
	return k.mount(SourceProc, target, FstypeProc, MS_NOSUID|MS_NOEXEC|MS_NODEV, zeroString)
}

func (p *MountProcOp) Is(op Op) bool {
	vp, ok := op.(*MountProcOp)
	return ok && p.Valid() && vp.Valid() &&
		p.Target.Is(vp.Target)
}
func (*MountProcOp) prefix() string   { return "mounting" }
func (p *MountProcOp) String() string { return fmt.Sprintf("proc on %q", p.Target) }
