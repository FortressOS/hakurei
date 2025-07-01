package sandbox

import (
	"encoding/gob"
	"fmt"
	"math"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"unsafe"
)

type (
	Ops []Op
	Op  interface {
		// early is called in host root.
		early(params *Params) error
		// apply is called in intermediate root.
		apply(params *Params) error

		prefix() string
		Is(op Op) bool
		fmt.Stringer
	}
)

func (f *Ops) Grow(n int) { *f = slices.Grow(*f, n) }

func init() { gob.Register(new(BindMountOp)) }

// BindMountOp bind mounts host path Source on container path Target.
type BindMountOp struct {
	Source, SourceFinal, Target string

	Flags int
}

const (
	BindOptional = 1 << iota
	BindWritable
	BindDevice
)

func (b *BindMountOp) early(*Params) error {
	if !path.IsAbs(b.Source) {
		return msg.WrapErr(syscall.EBADE,
			fmt.Sprintf("path %q is not absolute", b.Source))
	}

	if v, err := filepath.EvalSymlinks(b.Source); err != nil {
		if os.IsNotExist(err) && b.Flags&BindOptional != 0 {
			b.SourceFinal = "\x00"
			return nil
		}
		return wrapErrSelf(err)
	} else {
		b.SourceFinal = v
		return nil
	}
}

func (b *BindMountOp) apply(*Params) error {
	if b.SourceFinal == "\x00" {
		if b.Flags&BindOptional == 0 {
			// unreachable
			return syscall.EBADE
		}
		return nil
	}

	if !path.IsAbs(b.SourceFinal) || !path.IsAbs(b.Target) {
		return msg.WrapErr(syscall.EBADE,
			"path is not absolute")
	}

	source := toHost(b.SourceFinal)
	target := toSysroot(b.Target)

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

	var flags uintptr = syscall.MS_REC
	if b.Flags&BindWritable == 0 {
		flags |= syscall.MS_RDONLY
	}
	if b.Flags&BindDevice == 0 {
		flags |= syscall.MS_NODEV
	}

	return hostProc.bindMount(source, target, flags, b.SourceFinal == b.Target)
}

func (b *BindMountOp) Is(op Op) bool { vb, ok := op.(*BindMountOp); return ok && *b == *vb }
func (*BindMountOp) prefix() string  { return "mounting" }
func (b *BindMountOp) String() string {
	if b.Source == b.Target {
		return fmt.Sprintf("%q flags %#x", b.Source, b.Flags)
	}
	return fmt.Sprintf("%q on %q flags %#x", b.Source, b.Target, b.Flags&BindWritable)
}
func (f *Ops) Bind(source, target string, flags int) *Ops {
	*f = append(*f, &BindMountOp{source, "", target, flags})
	return f
}

func init() { gob.Register(new(MountProcOp)) }

// MountProcOp mounts a private instance of proc.
type MountProcOp string

func (p MountProcOp) early(*Params) error { return nil }
func (p MountProcOp) apply(params *Params) error {
	v := string(p)

	if !path.IsAbs(v) {
		return msg.WrapErr(syscall.EBADE,
			fmt.Sprintf("path %q is not absolute", v))
	}

	target := toSysroot(v)
	if err := os.MkdirAll(target, params.ParentPerm); err != nil {
		return wrapErrSelf(err)
	}
	return wrapErrSuffix(syscall.Mount("proc", target, "proc",
		syscall.MS_NOSUID|syscall.MS_NOEXEC|syscall.MS_NODEV, ""),
		fmt.Sprintf("cannot mount proc on %q:", v))
}

func (p MountProcOp) Is(op Op) bool  { vp, ok := op.(MountProcOp); return ok && p == vp }
func (MountProcOp) prefix() string   { return "mounting" }
func (p MountProcOp) String() string { return fmt.Sprintf("proc on %q", string(p)) }
func (f *Ops) Proc(dest string) *Ops {
	*f = append(*f, MountProcOp(dest))
	return f
}

func init() { gob.Register(new(MountDevOp)) }

// MountDevOp mounts part of host dev.
type MountDevOp string

func (d MountDevOp) early(*Params) error { return nil }
func (d MountDevOp) apply(params *Params) error {
	v := string(d)

	if !path.IsAbs(v) {
		return msg.WrapErr(syscall.EBADE,
			fmt.Sprintf("path %q is not absolute", v))
	}
	target := toSysroot(v)

	if err := mountTmpfs("devtmpfs", v, 0, params.ParentPerm); err != nil {
		return err
	}

	for _, name := range []string{"null", "zero", "full", "random", "urandom", "tty"} {
		targetPath := toSysroot(path.Join(v, name))
		if err := ensureFile(targetPath, 0444, params.ParentPerm); err != nil {
			return err
		}
		if err := hostProc.bindMount(
			toHost("/dev/"+name),
			targetPath,
			0,
			true,
		); err != nil {
			return err
		}
	}
	for i, name := range []string{"stdin", "stdout", "stderr"} {
		if err := os.Symlink(
			"/proc/self/fd/"+string(rune(i+'0')),
			path.Join(target, name),
		); err != nil {
			return wrapErrSelf(err)
		}
	}
	for _, pair := range [][2]string{
		{"/proc/self/fd", "fd"},
		{"/proc/kcore", "core"},
		{"pts/ptmx", "ptmx"},
	} {
		if err := os.Symlink(pair[0], path.Join(target, pair[1])); err != nil {
			return wrapErrSelf(err)
		}
	}

	devPtsPath := path.Join(target, "pts")
	for _, name := range []string{path.Join(target, "shm"), devPtsPath} {
		if err := os.Mkdir(name, params.ParentPerm); err != nil {
			return wrapErrSelf(err)
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
			consolePath := toSysroot(path.Join(v, "console"))
			if err := ensureFile(consolePath, 0444, params.ParentPerm); err != nil {
				return err
			}
			if name, err := os.Readlink(hostProc.stdout()); err != nil {
				return wrapErrSelf(err)
			} else if err = hostProc.bindMount(
				toHost(name),
				consolePath,
				0,
				false,
			); err != nil {
				return err
			}
		}
	}

	return nil
}

func (d MountDevOp) Is(op Op) bool  { vd, ok := op.(MountDevOp); return ok && d == vd }
func (MountDevOp) prefix() string   { return "mounting" }
func (d MountDevOp) String() string { return fmt.Sprintf("dev on %q", string(d)) }
func (f *Ops) Dev(dest string) *Ops {
	*f = append(*f, MountDevOp(dest))
	return f
}

func init() { gob.Register(new(MountMqueueOp)) }

// MountMqueueOp mounts a private mqueue instance on container Path.
type MountMqueueOp string

func (m MountMqueueOp) early(*Params) error { return nil }
func (m MountMqueueOp) apply(params *Params) error {
	v := string(m)

	if !path.IsAbs(v) {
		return msg.WrapErr(syscall.EBADE,
			fmt.Sprintf("path %q is not absolute", v))
	}

	target := toSysroot(v)
	if err := os.MkdirAll(target, params.ParentPerm); err != nil {
		return wrapErrSelf(err)
	}
	return wrapErrSuffix(syscall.Mount("mqueue", target, "mqueue",
		syscall.MS_NOSUID|syscall.MS_NOEXEC|syscall.MS_NODEV, ""),
		fmt.Sprintf("cannot mount mqueue on %q:", v))
}

func (m MountMqueueOp) Is(op Op) bool  { vm, ok := op.(MountMqueueOp); return ok && m == vm }
func (MountMqueueOp) prefix() string   { return "mounting" }
func (m MountMqueueOp) String() string { return fmt.Sprintf("mqueue on %q", string(m)) }
func (f *Ops) Mqueue(dest string) *Ops {
	*f = append(*f, MountMqueueOp(dest))
	return f
}

func init() { gob.Register(new(MountTmpfsOp)) }

// MountTmpfsOp mounts tmpfs on container Path.
type MountTmpfsOp struct {
	Path string
	Size int
	Perm os.FileMode
}

func (t *MountTmpfsOp) early(*Params) error { return nil }
func (t *MountTmpfsOp) apply(*Params) error {
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

func (t *MountTmpfsOp) Is(op Op) bool  { vt, ok := op.(*MountTmpfsOp); return ok && *t == *vt }
func (*MountTmpfsOp) prefix() string   { return "mounting" }
func (t *MountTmpfsOp) String() string { return fmt.Sprintf("tmpfs on %q size %d", t.Path, t.Size) }
func (f *Ops) Tmpfs(dest string, size int, perm os.FileMode) *Ops {
	*f = append(*f, &MountTmpfsOp{dest, size, perm})
	return f
}

func init() { gob.Register(new(SymlinkOp)) }

// SymlinkOp creates a symlink in the container filesystem.
type SymlinkOp [2]string

func (l *SymlinkOp) early(*Params) error {
	if strings.HasPrefix(l[0], "*") {
		l[0] = l[0][1:]
		if !path.IsAbs(l[0]) {
			return msg.WrapErr(syscall.EBADE,
				fmt.Sprintf("path %q is not absolute", l[0]))
		}
		if name, err := os.Readlink(l[0]); err != nil {
			return wrapErrSelf(err)
		} else {
			l[0] = name
		}
	}
	return nil
}
func (l *SymlinkOp) apply(params *Params) error {
	// symlink target is an arbitrary path value, so only validate link name here
	if !path.IsAbs(l[1]) {
		return msg.WrapErr(syscall.EBADE,
			fmt.Sprintf("path %q is not absolute", l[1]))
	}

	target := toSysroot(l[1])
	if err := os.MkdirAll(path.Dir(target), params.ParentPerm); err != nil {
		return wrapErrSelf(err)
	}
	if err := os.Symlink(l[0], target); err != nil {
		return wrapErrSelf(err)
	}
	return nil
}

func (l *SymlinkOp) Is(op Op) bool  { vl, ok := op.(*SymlinkOp); return ok && *l == *vl }
func (*SymlinkOp) prefix() string   { return "creating" }
func (l *SymlinkOp) String() string { return fmt.Sprintf("symlink on %q target %q", l[1], l[0]) }
func (f *Ops) Link(target, linkName string) *Ops {
	*f = append(*f, &SymlinkOp{target, linkName})
	return f
}

func init() { gob.Register(new(MkdirOp)) }

// MkdirOp creates a directory in the container filesystem.
type MkdirOp struct {
	Path string
	Perm os.FileMode
}

func (m *MkdirOp) early(*Params) error { return nil }
func (m *MkdirOp) apply(*Params) error {
	if !path.IsAbs(m.Path) {
		return msg.WrapErr(syscall.EBADE,
			fmt.Sprintf("path %q is not absolute", m.Path))
	}

	if err := os.MkdirAll(toSysroot(m.Path), m.Perm); err != nil {
		return wrapErrSelf(err)
	}
	return nil
}

func (m *MkdirOp) Is(op Op) bool  { vm, ok := op.(*MkdirOp); return ok && m == vm }
func (*MkdirOp) prefix() string   { return "creating" }
func (m *MkdirOp) String() string { return fmt.Sprintf("directory %q perm %s", m.Path, m.Perm) }
func (f *Ops) Mkdir(dest string, perm os.FileMode) *Ops {
	*f = append(*f, &MkdirOp{dest, perm})
	return f
}

func init() { gob.Register(new(TmpfileOp)) }

// TmpfileOp places a file in container Path containing Data.
type TmpfileOp struct {
	Path string
	Data []byte
}

func (t *TmpfileOp) early(*Params) error { return nil }
func (t *TmpfileOp) apply(params *Params) error {
	if !path.IsAbs(t.Path) {
		return msg.WrapErr(syscall.EBADE,
			fmt.Sprintf("path %q is not absolute", t.Path))
	}

	var tmpPath string
	if f, err := os.CreateTemp("/", "tmp.*"); err != nil {
		return wrapErrSelf(err)
	} else if _, err = f.Write(t.Data); err != nil {
		return wrapErrSuffix(err,
			"cannot write to intermediate file:")
	} else if err = f.Close(); err != nil {
		return wrapErrSuffix(err,
			"cannot close intermediate file:")
	} else {
		tmpPath = f.Name()
	}

	target := toSysroot(t.Path)
	if err := ensureFile(target, 0444, params.ParentPerm); err != nil {
		return err
	} else if err = hostProc.bindMount(
		tmpPath,
		target,
		syscall.MS_RDONLY|syscall.MS_NODEV,
		false,
	); err != nil {
		return err
	} else if err = os.Remove(tmpPath); err != nil {
		return wrapErrSelf(err)
	}
	return nil
}

func (t *TmpfileOp) Is(op Op) bool {
	vt, ok := op.(*TmpfileOp)
	return ok && t.Path == vt.Path && slices.Equal(t.Data, vt.Data)
}
func (*TmpfileOp) prefix() string { return "placing" }
func (t *TmpfileOp) String() string {
	return fmt.Sprintf("tmpfile %q (%d bytes)", t.Path, len(t.Data))
}
func (f *Ops) Place(name string, data []byte) *Ops { *f = append(*f, &TmpfileOp{name, data}); return f }
func (f *Ops) PlaceP(name string, dataP **[]byte) *Ops {
	t := &TmpfileOp{Path: name}
	*dataP = &t.Data

	*f = append(*f, t)
	return f
}

func init() { gob.Register(new(AutoEtcOp)) }

// AutoEtcOp expands host /etc into a toplevel symlink mirror with /etc semantics.
// This is not a generic setup op. It is implemented here to reduce ipc overhead.
type AutoEtcOp struct{ Prefix string }

func (e *AutoEtcOp) early(*Params) error { return nil }
func (e *AutoEtcOp) apply(*Params) error {
	const target = sysrootPath + "/etc/"
	rel := e.hostRel() + "/"

	if err := os.MkdirAll(target, 0755); err != nil {
		return wrapErrSelf(err)
	}
	if d, err := os.ReadDir(toSysroot(e.hostPath())); err != nil {
		return wrapErrSelf(err)
	} else {
		for _, ent := range d {
			n := ent.Name()
			switch n {
			case ".host":

			case "passwd":
			case "group":

			case "mtab":
				if err = os.Symlink("/proc/mounts", target+n); err != nil {
					return wrapErrSelf(err)
				}

			default:
				if err = os.Symlink(rel+n, target+n); err != nil {
					return wrapErrSelf(err)
				}
			}
		}
	}

	return nil
}
func (e *AutoEtcOp) hostPath() string { return "/etc/" + e.hostRel() }
func (e *AutoEtcOp) hostRel() string  { return ".host/" + e.Prefix }

func (e *AutoEtcOp) Is(op Op) bool {
	ve, ok := op.(*AutoEtcOp)
	return ok && ((e == nil && ve == nil) || (e != nil && ve != nil && *e == *ve))
}
func (*AutoEtcOp) prefix() string   { return "setting up" }
func (e *AutoEtcOp) String() string { return fmt.Sprintf("auto etc %s", e.Prefix) }
func (f *Ops) Etc(host, prefix string) *Ops {
	e := &AutoEtcOp{prefix}
	f.Mkdir("/etc", 0755)
	f.Bind(host, e.hostPath(), 0)
	*f = append(*f, e)
	return f
}
