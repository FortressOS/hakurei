package sandbox

import (
	"encoding/gob"
	"fmt"
	"math"
	"os"
	"path"
	"syscall"
	"unsafe"
)

func init() { gob.Register(new(BindMount)) }

// BindMount bind mounts host path Source on container path Target.
type BindMount struct {
	Source, Target string

	Flags int
}

func (b *BindMount) apply(*Params) error {
	if !path.IsAbs(b.Source) || !path.IsAbs(b.Target) {
		return msg.WrapErr(syscall.EBADE,
			"path is not absolute")
	}
	return bindMount(b.Source, b.Target, b.Flags)
}

func (b *BindMount) Is(op Op) bool { vb, ok := op.(*BindMount); return ok && *b == *vb }
func (*BindMount) prefix() string  { return "mounting" }
func (b *BindMount) String() string {
	if b.Source == b.Target {
		return fmt.Sprintf("%q flags %#x", b.Source, b.Flags)
	}
	return fmt.Sprintf("%q on %q flags %#x", b.Source, b.Target, b.Flags&BindWritable)
}
func (f *Ops) Bind(source, target string, flags int) *Ops {
	*f = append(*f, &BindMount{source, target, flags | bindRecursive})
	return f
}

func init() { gob.Register(new(MountProc)) }

// MountProc mounts a private instance of proc.
type MountProc string

func (p MountProc) apply(*Params) error {
	v := string(p)

	if !path.IsAbs(v) {
		return msg.WrapErr(syscall.EBADE,
			fmt.Sprintf("path %q is not absolute", v))
	}

	target := toSysroot(v)
	if err := os.MkdirAll(target, 0755); err != nil {
		return msg.WrapErr(err, err.Error())
	}
	return wrapErrSuffix(syscall.Mount("proc", target, "proc",
		syscall.MS_NOSUID|syscall.MS_NOEXEC|syscall.MS_NODEV, ""),
		fmt.Sprintf("cannot mount proc on %q:", v))
}

func (p MountProc) Is(op Op) bool  { vp, ok := op.(MountProc); return ok && p == vp }
func (MountProc) prefix() string   { return "mounting" }
func (p MountProc) String() string { return fmt.Sprintf("proc on %q", string(p)) }
func (f *Ops) Proc(dest string) *Ops {
	*f = append(*f, MountProc(dest))
	return f
}

func init() { gob.Register(new(MountDev)) }

// MountDev mounts part of host dev.
type MountDev string

func (d MountDev) apply(params *Params) error {
	v := string(d)

	if !path.IsAbs(v) {
		return msg.WrapErr(syscall.EBADE,
			fmt.Sprintf("path %q is not absolute", v))
	}
	target := toSysroot(v)

	if err := mountTmpfs("devtmpfs", v, 0, 0755); err != nil {
		return err
	}

	for _, name := range []string{"null", "zero", "full", "random", "urandom", "tty"} {
		if err := bindMount(
			"/dev/"+name, path.Join(v, name),
			BindSource|BindDevice,
		); err != nil {
			return err
		}
	}
	for i, name := range []string{"stdin", "stdout", "stderr"} {
		if err := os.Symlink(
			"/proc/self/fd/"+string(rune(i+'0')),
			path.Join(target, name),
		); err != nil {
			return msg.WrapErr(err, err.Error())
		}
	}
	for _, pair := range [][2]string{
		{"/proc/self/fd", "fd"},
		{"/proc/kcore", "core"},
		{"pts/ptmx", "ptmx"},
	} {
		if err := os.Symlink(pair[0], path.Join(target, pair[1])); err != nil {
			return msg.WrapErr(err, err.Error())
		}
	}

	devPtsPath := path.Join(target, "pts")
	for _, name := range []string{path.Join(target, "shm"), devPtsPath} {
		if err := os.Mkdir(name, 0755); err != nil {
			return msg.WrapErr(err, err.Error())
		}
	}

	if err := syscall.Mount("devpts", devPtsPath, "devpts",
		syscall.MS_NOSUID|syscall.MS_NOEXEC,
		"newinstance,ptmxmode=0666,mode=620"); err != nil {
		return wrapErrSuffix(err,
			fmt.Sprintf("cannot mount devpts on %q:", devPtsPath))
	}

	if params.Flags&FAllowTTY != 0 {
		var buf [8]byte
		if _, _, errno := syscall.Syscall(
			syscall.SYS_IOCTL, 1, syscall.TIOCGWINSZ,
			uintptr(unsafe.Pointer(&buf[0])),
		); errno == 0 {
			if err := bindMount("/proc/self/fd/1", path.Join(v, "console"), BindDevice); err != nil {
				return err
			}
		}
	}

	return nil
}

func (d MountDev) Is(op Op) bool  { vd, ok := op.(MountDev); return ok && d == vd }
func (MountDev) prefix() string   { return "mounting" }
func (d MountDev) String() string { return fmt.Sprintf("dev on %q", string(d)) }
func (f *Ops) Dev(dest string) *Ops {
	*f = append(*f, MountDev(dest))
	return f
}

func init() { gob.Register(new(MountMqueue)) }

// MountMqueue mounts a private mqueue instance on container Path.
type MountMqueue string

func (m MountMqueue) apply(*Params) error {
	v := string(m)

	if !path.IsAbs(v) {
		return msg.WrapErr(syscall.EBADE,
			fmt.Sprintf("path %q is not absolute", v))
	}

	target := toSysroot(v)
	if err := os.MkdirAll(target, 0755); err != nil {
		return msg.WrapErr(err, err.Error())
	}
	return wrapErrSuffix(syscall.Mount("mqueue", target, "mqueue",
		syscall.MS_NOSUID|syscall.MS_NOEXEC|syscall.MS_NODEV, ""),
		fmt.Sprintf("cannot mount mqueue on %q:", v))
}

func (m MountMqueue) Is(op Op) bool  { vm, ok := op.(MountMqueue); return ok && m == vm }
func (MountMqueue) prefix() string   { return "mounting" }
func (m MountMqueue) String() string { return fmt.Sprintf("mqueue on %q", string(m)) }
func (f *Ops) Mqueue(dest string) *Ops {
	*f = append(*f, MountMqueue(dest))
	return f
}

func init() { gob.Register(new(MountTmpfs)) }

// MountTmpfs mounts tmpfs on container Path.
type MountTmpfs struct {
	Path string
	Size int
	Perm os.FileMode
}

func (t *MountTmpfs) apply(*Params) error {
	if !path.IsAbs(t.Path) {
		return msg.WrapErr(syscall.EBADE,
			fmt.Sprintf("path %q is not absolute", t.Path))
	}
	if t.Size < 0 || t.Size > math.MaxUint>>1 {
		return msg.WrapErr(syscall.EBADE,
			fmt.Sprintf("size %d out of bounds", t.Size))
	}
	return mountTmpfs("tmpfs", t.Path, t.Size, t.Perm)
}

func (t *MountTmpfs) Is(op Op) bool  { vt, ok := op.(*MountTmpfs); return ok && *t == *vt }
func (*MountTmpfs) prefix() string   { return "mounting" }
func (t *MountTmpfs) String() string { return fmt.Sprintf("tmpfs on %q size %d", t.Path, t.Size) }
func (f *Ops) Tmpfs(dest string, size int, perm os.FileMode) *Ops {
	*f = append(*f, &MountTmpfs{dest, size, perm})
	return f
}

func init() { gob.Register(new(Symlink)) }

// Symlink creates a symlink in the container filesystem.
type Symlink [2]string

func (l *Symlink) apply(*Params) error {
	// symlink target is an arbitrary path value, so only validate link name here
	if !path.IsAbs(l[1]) {
		return msg.WrapErr(syscall.EBADE,
			fmt.Sprintf("path %q is not absolute", l[1]))
	}
	if err := os.Symlink(l[0], toSysroot(l[1])); err != nil {
		return msg.WrapErr(err, err.Error())
	}
	return nil
}

func (l *Symlink) Is(op Op) bool  { vl, ok := op.(*Symlink); return ok && *l == *vl }
func (*Symlink) prefix() string   { return "creating" }
func (l *Symlink) String() string { return fmt.Sprintf("symlink on %q target %q", l[1], l[0]) }
func (f *Ops) Link(target, linkName string) *Ops {
	*f = append(*f, &Symlink{target, linkName})
	return f
}
