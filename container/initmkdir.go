package container

import (
	"encoding/gob"
	"fmt"
	"os"
	"syscall"
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

func (m *MkdirOp) early(*setupState) error { return nil }
func (m *MkdirOp) apply(*setupState) error {
	if m.Path == nil {
		return syscall.EBADE
	}
	return wrapErrSelf(os.MkdirAll(toSysroot(m.Path.String()), m.Perm))
}

func (m *MkdirOp) Is(op Op) bool {
	vm, ok := op.(*MkdirOp)
	return ok && ((m == nil && vm == nil) || (m != nil && vm != nil &&
		m.Path != nil && vm.Path != nil &&
		m.Path.String() == vm.Path.String() &&
		m.Perm == vm.Perm))
}
func (*MkdirOp) prefix() string   { return "creating" }
func (m *MkdirOp) String() string { return fmt.Sprintf("directory %q perm %s", m.Path, m.Perm) }
