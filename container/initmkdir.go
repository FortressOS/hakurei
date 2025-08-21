package container

import (
	"encoding/gob"
	"fmt"
	"os"
)

func init() { gob.Register(new(MkdirOp)) }

// Mkdir appends an [Op] that creates a directory in the container filesystem.
func (f *Ops) Mkdir(name *Absolute, perm os.FileMode) *Ops {
	*f = append(*f, &MkdirOp{name, perm})
	return f
}

// MkdirOp creates a directory at container Path with permission bits set to Perm.
type MkdirOp struct {
	Path *Absolute
	Perm os.FileMode
}

func (m *MkdirOp) Valid() bool                                { return m != nil && m.Path != nil }
func (m *MkdirOp) early(*setupState, syscallDispatcher) error { return nil }
func (m *MkdirOp) apply(_ *setupState, k syscallDispatcher) error {
	return wrapErrSelf(k.mkdirAll(toSysroot(m.Path.String()), m.Perm))
}

func (m *MkdirOp) Is(op Op) bool {
	vm, ok := op.(*MkdirOp)
	return ok && m.Valid() && vm.Valid() &&
		m.Path.Is(vm.Path) &&
		m.Perm == vm.Perm
}
func (*MkdirOp) prefix() string   { return "creating" }
func (m *MkdirOp) String() string { return fmt.Sprintf("directory %q perm %s", m.Path, m.Perm) }
