package sandbox

import (
	"encoding/gob"
	"fmt"
	"math"
	"os"
	"path"
	"syscall"

	"git.gensokyo.uk/security/fortify/internal/fmsg"
)

func init() { gob.Register(new(BindMount)) }

// BindMount bind mounts host path Source on container path Target.
type BindMount struct {
	Source, Target string

	Flags int
}

func (b *BindMount) apply() error {
	if !path.IsAbs(b.Source) || !path.IsAbs(b.Target) {
		return fmsg.WrapError(syscall.EBADE,
			"path is not absolute")
	}
	return bindMount(b.Source, b.Target, b.Flags)
}

func (b *BindMount) Is(op Op) bool { vb, ok := op.(*BindMount); return ok && *b == *vb }
func (b *BindMount) String() string {
	if b.Source == b.Target {
		return fmt.Sprintf("%q flags %#x", b.Source, b.Flags)
	}
	return fmt.Sprintf("%q on %q flags %#x", b.Source, b.Target, b.Flags&BindWritable)
}
func (f *Ops) Bind(source, target string, flags int) *Ops {
	*f = append(*f, &BindMount{source, target, flags | BindRecursive})
	return f
}

func init() { gob.Register(new(MountProc)) }

// MountProc mounts a private proc instance on container Path.
type MountProc struct {
	Path string
}

func (p *MountProc) apply() error {
	if !path.IsAbs(p.Path) {
		return fmsg.WrapError(syscall.EBADE,
			fmt.Sprintf("path %q is not absolute", p.Path))
	}

	target := toSysroot(p.Path)
	if err := os.MkdirAll(target, 0755); err != nil {
		return fmsg.WrapError(err, err.Error())
	}
	return fmsg.WrapErrorSuffix(syscall.Mount("proc", target, "proc",
		syscall.MS_NOSUID|syscall.MS_NOEXEC|syscall.MS_NODEV, ""),
		fmt.Sprintf("cannot mount proc on %q:", p.Path))
}

func (p *MountProc) Is(op Op) bool  { vp, ok := op.(*MountProc); return ok && *p == *vp }
func (p *MountProc) String() string { return fmt.Sprintf("proc on %q", p.Path) }
func (f *Ops) Proc(dest string) *Ops {
	*f = append(*f, &MountProc{dest})
	return f
}

func init() { gob.Register(new(MountTmpfs)) }

// MountTmpfs mounts tmpfs on container Path.
type MountTmpfs struct {
	Path string
	Size int
	Mode os.FileMode
}

func (t *MountTmpfs) apply() error {
	if !path.IsAbs(t.Path) {
		return fmsg.WrapError(syscall.EBADE,
			fmt.Sprintf("path %q is not absolute", t.Path))
	}
	if t.Size < 0 || t.Size > math.MaxUint>>1 {
		return fmsg.WrapError(syscall.EBADE,
			fmt.Sprintf("size %d out of bounds", t.Size))
	}
	target := toSysroot(t.Path)
	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}
	opt := fmt.Sprintf("mode=%#o", t.Mode)
	if t.Size > 0 {
		opt += fmt.Sprintf(",size=%d", t.Mode)
	}
	return fmsg.WrapErrorSuffix(syscall.Mount("tmpfs", target, "tmpfs",
		syscall.MS_NOSUID|syscall.MS_NODEV, opt),
		fmt.Sprintf("cannot mount tmpfs on %q:", t.Path))
}

func (t *MountTmpfs) Is(op Op) bool  { vt, ok := op.(*MountTmpfs); return ok && *t == *vt }
func (t *MountTmpfs) String() string { return fmt.Sprintf("tmpfs on %q size %d", t.Path, t.Size) }
func (f *Ops) Tmpfs(dest string, size int, mode os.FileMode) *Ops {
	*f = append(*f, &MountTmpfs{dest, size, mode})
	return f
}
