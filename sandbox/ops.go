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

func init() { gob.Register(new(BindMount)) }

// BindMount bind mounts host path Source on container path Target.
type BindMount struct {
	Source, SourceFinal, Target string

	Flags int
}

const (
	BindOptional = 1 << iota
	BindWritable
	BindDevice
)

func (b *BindMount) early(*Params) error {
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

func (b *BindMount) apply(*Params) error {
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

func (b *BindMount) Is(op Op) bool { vb, ok := op.(*BindMount); return ok && *b == *vb }
func (*BindMount) prefix() string  { return "mounting" }
func (b *BindMount) String() string {
	if b.Source == b.Target {
		return fmt.Sprintf("%q flags %#x", b.Source, b.Flags)
	}
	return fmt.Sprintf("%q on %q flags %#x", b.Source, b.Target, b.Flags&BindWritable)
}
func (f *Ops) Bind(source, target string, flags int) *Ops {
	*f = append(*f, &BindMount{source, "", target, flags})
	return f
}

func init() { gob.Register(new(MountProc)) }

// MountProc mounts a private instance of proc.
type MountProc string

func (p MountProc) early(*Params) error { return nil }
func (p MountProc) apply(params *Params) error {
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

func (d MountDev) early(*Params) error { return nil }
func (d MountDev) apply(params *Params) error {
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

func (m MountMqueue) early(*Params) error { return nil }
func (m MountMqueue) apply(params *Params) error {
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

func (t *MountTmpfs) early(*Params) error { return nil }
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

func (l *Symlink) early(*Params) error {
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
func (l *Symlink) apply(params *Params) error {
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

func (l *Symlink) Is(op Op) bool  { vl, ok := op.(*Symlink); return ok && *l == *vl }
func (*Symlink) prefix() string   { return "creating" }
func (l *Symlink) String() string { return fmt.Sprintf("symlink on %q target %q", l[1], l[0]) }
func (f *Ops) Link(target, linkName string) *Ops {
	*f = append(*f, &Symlink{target, linkName})
	return f
}

func init() { gob.Register(new(Mkdir)) }

// Mkdir creates a directory in the container filesystem.
type Mkdir struct {
	Path string
	Perm os.FileMode
}

func (m *Mkdir) early(*Params) error { return nil }
func (m *Mkdir) apply(*Params) error {
	if !path.IsAbs(m.Path) {
		return msg.WrapErr(syscall.EBADE,
			fmt.Sprintf("path %q is not absolute", m.Path))
	}

	if err := os.MkdirAll(toSysroot(m.Path), m.Perm); err != nil {
		return wrapErrSelf(err)
	}
	return nil
}

func (m *Mkdir) Is(op Op) bool  { vm, ok := op.(*Mkdir); return ok && m == vm }
func (*Mkdir) prefix() string   { return "creating" }
func (m *Mkdir) String() string { return fmt.Sprintf("directory %q perm %s", m.Path, m.Perm) }
func (f *Ops) Mkdir(dest string, perm os.FileMode) *Ops {
	*f = append(*f, &Mkdir{dest, perm})
	return f
}

func init() { gob.Register(new(Tmpfile)) }

// Tmpfile places a file in container Path containing Data.
type Tmpfile struct {
	Path string
	Data []byte
}

func (t *Tmpfile) early(*Params) error { return nil }
func (t *Tmpfile) apply(params *Params) error {
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

func (t *Tmpfile) Is(op Op) bool {
	vt, ok := op.(*Tmpfile)
	return ok && t.Path == vt.Path && slices.Equal(t.Data, vt.Data)
}
func (*Tmpfile) prefix() string { return "placing" }
func (t *Tmpfile) String() string {
	return fmt.Sprintf("tmpfile %q (%d bytes)", t.Path, len(t.Data))
}
func (f *Ops) Place(name string, data []byte) *Ops { *f = append(*f, &Tmpfile{name, data}); return f }
func (f *Ops) PlaceP(name string, dataP **[]byte) *Ops {
	t := &Tmpfile{Path: name}
	*dataP = &t.Data

	*f = append(*f, t)
	return f
}

func init() { gob.Register(new(AutoEtc)) }

// AutoEtc expands host /etc into a toplevel symlink mirror with /etc semantics.
// This is not a generic setup op. It is implemented here to reduce ipc overhead.
type AutoEtc struct{ Prefix string }

func (e *AutoEtc) early(*Params) error { return nil }
func (e *AutoEtc) apply(*Params) error {
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
func (e *AutoEtc) hostPath() string { return "/etc/" + e.hostRel() }
func (e *AutoEtc) hostRel() string  { return ".host/" + e.Prefix }

func (e *AutoEtc) Is(op Op) bool {
	ve, ok := op.(*AutoEtc)
	return ok && ((e == nil && ve == nil) || (e != nil && ve != nil && *e == *ve))
}
func (*AutoEtc) prefix() string   { return "setting up" }
func (e *AutoEtc) String() string { return fmt.Sprintf("auto etc %s", e.Prefix) }
func (f *Ops) Etc(host, prefix string) *Ops {
	e := &AutoEtc{prefix}
	f.Mkdir("/etc", 0755)
	f.Bind(host, e.hostPath(), 0)
	*f = append(*f, e)
	return f
}
