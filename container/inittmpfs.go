package container

import (
	"encoding/gob"
	"fmt"
	"math"
	"os"
	"strconv"
	. "syscall"
)

func init() { gob.Register(new(MountTmpfsOp)) }

type TmpfsSizeError int

func (e TmpfsSizeError) Error() string {
	return "tmpfs size " + strconv.Itoa(int(e)) + " out of bounds"
}

// Tmpfs appends an [Op] that mounts tmpfs on container path [MountTmpfsOp.Path].
func (f *Ops) Tmpfs(target *Absolute, size int, perm os.FileMode) *Ops {
	*f = append(*f, &MountTmpfsOp{SourceTmpfsEphemeral, target, MS_NOSUID | MS_NODEV, size, perm})
	return f
}

// Readonly appends an [Op] that mounts read-only tmpfs on container path [MountTmpfsOp.Path].
func (f *Ops) Readonly(target *Absolute, perm os.FileMode) *Ops {
	*f = append(*f, &MountTmpfsOp{SourceTmpfsReadonly, target, MS_RDONLY | MS_NOSUID | MS_NODEV, 0, perm})
	return f
}

// MountTmpfsOp mounts [FstypeTmpfs] on container Path.
type MountTmpfsOp struct {
	FSName string
	Path   *Absolute
	Flags  uintptr
	Size   int
	Perm   os.FileMode
}

func (t *MountTmpfsOp) Valid() bool                                { return t != nil && t.Path != nil && t.FSName != zeroString }
func (t *MountTmpfsOp) early(*setupState, syscallDispatcher) error { return nil }
func (t *MountTmpfsOp) apply(_ *setupState, k syscallDispatcher) error {
	if t.Size < 0 || t.Size > math.MaxUint>>1 {
		return TmpfsSizeError(t.Size)
	}
	return k.mountTmpfs(t.FSName, toSysroot(t.Path.String()), t.Flags, t.Size, t.Perm)
}

func (t *MountTmpfsOp) Is(op Op) bool {
	vt, ok := op.(*MountTmpfsOp)
	return ok && t.Valid() && vt.Valid() &&
		t.FSName == vt.FSName &&
		t.Path.Is(vt.Path) &&
		t.Flags == vt.Flags &&
		t.Size == vt.Size &&
		t.Perm == vt.Perm
}
func (*MountTmpfsOp) prefix() (string, bool) { return "mounting", true }
func (t *MountTmpfsOp) String() string       { return fmt.Sprintf("tmpfs on %q size %d", t.Path, t.Size) }
