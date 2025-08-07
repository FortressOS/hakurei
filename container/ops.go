package container

import (
	"encoding/gob"
	"fmt"
	"math"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	. "syscall"
	"unsafe"
)

const (
	// intermediate root file name pattern for [MountOverlayOp.Upper];
	// remains after apply returns
	intermediatePatternOverlayUpper = "overlay.upper.*"
	// intermediate root file name pattern for [MountOverlayOp.Work];
	// remains after apply returns
	intermediatePatternOverlayWork = "overlay.work.*"
	// intermediate root file name pattern for [TmpfileOp]
	intermediatePatternTmpfile = "tmp.*"
)

type (
	Ops []Op

	// Op is a generic setup step ran inside the container init.
	// Implementations of this interface are sent as a stream of gobs.
	Op interface {
		// early is called in host root.
		early(params *Params) error
		// apply is called in intermediate root.
		apply(params *Params) error

		prefix() string
		Is(op Op) bool
		fmt.Stringer
	}
)

// Grow grows the slice Ops points to using [slices.Grow].
func (f *Ops) Grow(n int) { *f = slices.Grow(*f, n) }

func init() { gob.Register(new(RemountOp)) }

// Remount appends an [Op] that applies [RemountOp.Flags] on container path [RemountOp.Target].
func (f *Ops) Remount(target string, flags uintptr) *Ops {
	*f = append(*f, &RemountOp{target, flags})
	return f
}

type RemountOp struct {
	Target string
	Flags  uintptr
}

func (*RemountOp) early(*Params) error { return nil }
func (r *RemountOp) apply(*Params) error {
	if !path.IsAbs(r.Target) {
		return msg.WrapErr(EBADE, fmt.Sprintf("path %q is not absolute", r.Target))
	}
	return wrapErrSuffix(hostProc.remount(toSysroot(r.Target), r.Flags),
		fmt.Sprintf("cannot remount %q:", r.Target))
}

func (r *RemountOp) Is(op Op) bool  { vr, ok := op.(*RemountOp); return ok && *r == *vr }
func (*RemountOp) prefix() string   { return "remounting" }
func (r *RemountOp) String() string { return fmt.Sprintf("%q flags %#x", r.Target, r.Flags) }

func init() { gob.Register(new(BindMountOp)) }

// Bind appends an [Op] that bind mounts host path [BindMountOp.Source] on container path [BindMountOp.Target].
func (f *Ops) Bind(source, target string, flags int) *Ops {
	*f = append(*f, &BindMountOp{source, "", target, flags})
	return f
}

type BindMountOp struct {
	Source, sourceFinal, Target string

	Flags int
}

const (
	// BindOptional skips nonexistent host paths.
	BindOptional = 1 << iota
	// BindWritable mounts filesystem read-write.
	BindWritable
	// BindDevice allows access to devices (special files) on this filesystem.
	BindDevice
)

func (b *BindMountOp) early(*Params) error {
	if !path.IsAbs(b.Source) {
		return msg.WrapErr(EBADE, fmt.Sprintf("path %q is not absolute", b.Source))
	}

	if v, err := filepath.EvalSymlinks(b.Source); err != nil {
		if os.IsNotExist(err) && b.Flags&BindOptional != 0 {
			b.sourceFinal = "\x00"
			return nil
		}
		return wrapErrSelf(err)
	} else {
		b.sourceFinal = v
		return nil
	}
}

func (b *BindMountOp) apply(*Params) error {
	if b.sourceFinal == "\x00" {
		if b.Flags&BindOptional == 0 {
			// unreachable
			return EBADE
		}
		return nil
	}

	if !path.IsAbs(b.sourceFinal) || !path.IsAbs(b.Target) {
		return msg.WrapErr(EBADE, "path is not absolute")
	}

	source := toHost(b.sourceFinal)
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

	var flags uintptr = MS_REC
	if b.Flags&BindWritable == 0 {
		flags |= MS_RDONLY
	}
	if b.Flags&BindDevice == 0 {
		flags |= MS_NODEV
	}

	return hostProc.bindMount(source, target, flags, b.sourceFinal == b.Target)
}

func (b *BindMountOp) Is(op Op) bool { vb, ok := op.(*BindMountOp); return ok && *b == *vb }
func (*BindMountOp) prefix() string  { return "mounting" }
func (b *BindMountOp) String() string {
	if b.Source == b.Target {
		return fmt.Sprintf("%q flags %#x", b.Source, b.Flags)
	}
	return fmt.Sprintf("%q on %q flags %#x", b.Source, b.Target, b.Flags)
}

func init() { gob.Register(new(MountProcOp)) }

// Proc appends an [Op] that mounts a private instance of proc.
func (f *Ops) Proc(dest string) *Ops {
	*f = append(*f, MountProcOp(dest))
	return f
}

type MountProcOp string

func (p MountProcOp) early(*Params) error { return nil }
func (p MountProcOp) apply(params *Params) error {
	v := string(p)

	if !path.IsAbs(v) {
		return msg.WrapErr(EBADE, fmt.Sprintf("path %q is not absolute", v))
	}

	target := toSysroot(v)
	if err := os.MkdirAll(target, params.ParentPerm); err != nil {
		return wrapErrSelf(err)
	}
	return wrapErrSuffix(Mount(SourceProc, target, FstypeProc, MS_NOSUID|MS_NOEXEC|MS_NODEV, zeroString),
		fmt.Sprintf("cannot mount proc on %q:", v))
}

func (p MountProcOp) Is(op Op) bool  { vp, ok := op.(MountProcOp); return ok && p == vp }
func (MountProcOp) prefix() string   { return "mounting" }
func (p MountProcOp) String() string { return fmt.Sprintf("proc on %q", string(p)) }

func init() { gob.Register(new(MountDevOp)) }

// Dev appends an [Op] that mounts a subset of host /dev.
func (f *Ops) Dev(dest string, mqueue bool) *Ops {
	*f = append(*f, &MountDevOp{dest, mqueue, false})
	return f
}

// DevWritable appends an [Op] that mounts a writable subset of host /dev.
// There is usually no good reason to write to /dev, so this should always be followed by a [RemountOp].
func (f *Ops) DevWritable(dest string, mqueue bool) *Ops {
	*f = append(*f, &MountDevOp{dest, mqueue, true})
	return f
}

type MountDevOp struct {
	Target string
	Mqueue bool
	Write  bool
}

func (d *MountDevOp) early(*Params) error { return nil }
func (d *MountDevOp) apply(params *Params) error {
	if !path.IsAbs(d.Target) {
		return msg.WrapErr(EBADE, fmt.Sprintf("path %q is not absolute", d.Target))
	}
	target := toSysroot(d.Target)

	if err := mountTmpfs(SourceTmpfsDevtmpfs, target, MS_NOSUID|MS_NODEV, 0, params.ParentPerm); err != nil {
		return err
	}

	for _, name := range []string{"null", "zero", "full", "random", "urandom", "tty"} {
		targetPath := path.Join(target, name)
		if err := ensureFile(targetPath, 0444, params.ParentPerm); err != nil {
			return err
		}
		if err := hostProc.bindMount(
			toHost(FHSDev+name),
			targetPath,
			0,
			true,
		); err != nil {
			return err
		}
	}
	for i, name := range []string{"stdin", "stdout", "stderr"} {
		if err := os.Symlink(
			FHSProc+"self/fd/"+string(rune(i+'0')),
			path.Join(target, name),
		); err != nil {
			return wrapErrSelf(err)
		}
	}
	for _, pair := range [][2]string{
		{FHSProc + "self/fd", "fd"},
		{FHSProc + "kcore", "core"},
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

	if err := Mount(SourceDevpts, devPtsPath, FstypeDevpts, MS_NOSUID|MS_NOEXEC,
		"newinstance,ptmxmode=0666,mode=620"); err != nil {
		return wrapErrSuffix(err,
			fmt.Sprintf("cannot mount devpts on %q:", devPtsPath))
	}

	if params.RetainSession {
		var buf [8]byte
		if _, _, errno := Syscall(SYS_IOCTL, 1, TIOCGWINSZ, uintptr(unsafe.Pointer(&buf[0]))); errno == 0 {
			consolePath := path.Join(target, "console")
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

	if d.Mqueue {
		mqueueTarget := path.Join(target, "mqueue")
		if err := os.Mkdir(mqueueTarget, params.ParentPerm); err != nil {
			return wrapErrSelf(err)
		}
		if err := Mount(SourceMqueue, mqueueTarget, FstypeMqueue, MS_NOSUID|MS_NOEXEC|MS_NODEV, zeroString); err != nil {
			return wrapErrSuffix(err, "cannot mount mqueue:")
		}
	}

	if d.Write {
		return nil
	}
	return wrapErrSuffix(hostProc.remount(target, MS_RDONLY),
		fmt.Sprintf("cannot remount %q:", target))
}

func (d *MountDevOp) Is(op Op) bool { vd, ok := op.(*MountDevOp); return ok && *d == *vd }
func (*MountDevOp) prefix() string  { return "mounting" }
func (d *MountDevOp) String() string {
	if d.Mqueue {
		return fmt.Sprintf("dev on %q with mqueue", d.Target)
	}
	return fmt.Sprintf("dev on %q", d.Target)
}

func init() { gob.Register(new(MountTmpfsOp)) }

// Tmpfs appends an [Op] that mounts tmpfs on container path [MountTmpfsOp.Path].
func (f *Ops) Tmpfs(dest string, size int, perm os.FileMode) *Ops {
	*f = append(*f, &MountTmpfsOp{SourceTmpfsEphemeral, dest, MS_NOSUID | MS_NODEV, size, perm})
	return f
}

// Readonly appends an [Op] that mounts read-only tmpfs on container path [MountTmpfsOp.Path].
func (f *Ops) Readonly(dest string, perm os.FileMode) *Ops {
	*f = append(*f, &MountTmpfsOp{SourceTmpfsReadonly, dest, MS_RDONLY | MS_NOSUID | MS_NODEV, 0, perm})
	return f
}

type MountTmpfsOp struct {
	FSName string
	Path   string
	Flags  uintptr
	Size   int
	Perm   os.FileMode
}

func (t *MountTmpfsOp) early(*Params) error { return nil }
func (t *MountTmpfsOp) apply(*Params) error {
	if !path.IsAbs(t.Path) {
		return msg.WrapErr(EBADE, fmt.Sprintf("path %q is not absolute", t.Path))
	}
	if t.Size < 0 || t.Size > math.MaxUint>>1 {
		return msg.WrapErr(EBADE, fmt.Sprintf("size %d out of bounds", t.Size))
	}
	return mountTmpfs(t.FSName, toSysroot(t.Path), t.Flags, t.Size, t.Perm)
}

func (t *MountTmpfsOp) Is(op Op) bool  { vt, ok := op.(*MountTmpfsOp); return ok && *t == *vt }
func (*MountTmpfsOp) prefix() string   { return "mounting" }
func (t *MountTmpfsOp) String() string { return fmt.Sprintf("tmpfs on %q size %d", t.Path, t.Size) }

func init() { gob.Register(new(MountOverlayOp)) }

// Overlay appends an [Op] that mounts the overlay pseudo filesystem on [MountOverlayOp.Target].
func (f *Ops) Overlay(target, state, work string, layers ...string) *Ops {
	*f = append(*f, &MountOverlayOp{
		Target: target,
		Lower:  layers,
		Upper:  state,
		Work:   work,
	})
	return f
}

// OverlayEphemeral appends an [Op] that mounts the overlay pseudo filesystem on [MountOverlayOp.Target]
// with an ephemeral upperdir and workdir.
func (f *Ops) OverlayEphemeral(target string, layers ...string) *Ops {
	return f.Overlay(target, SourceTmpfsEphemeral, zeroString, layers...)
}

// OverlayReadonly appends an [Op] that mounts the overlay pseudo filesystem readonly on [MountOverlayOp.Target]
func (f *Ops) OverlayReadonly(target string, layers ...string) *Ops {
	return f.Overlay(target, zeroString, zeroString, layers...)
}

type MountOverlayOp struct {
	Target string

	// formatted for [OptionOverlayLowerdir], resolved, prefixed and escaped during early;
	Lower []string
	// formatted for [OptionOverlayUpperdir], resolved, prefixed and escaped during early;
	//
	// If Work is an empty string and Upper holds the special value [SourceTmpfsEphemeral],
	// an ephemeral upperdir and workdir will be set up.
	//
	// If both Work and Upper are empty strings, upperdir and workdir is omitted and the overlay is mounted readonly.
	Upper string
	// formatted for [OptionOverlayWorkdir], resolved, prefixed and escaped during early;
	Work string

	ephemeral bool
}

func (o *MountOverlayOp) early(*Params) error {
	if o.Work == zeroString {
		switch o.Upper {
		case SourceTmpfsEphemeral: // ephemeral
			o.ephemeral = true // intermediate root not yet available

		case zeroString: // readonly

		default:
			return msg.WrapErr(EINVAL, fmt.Sprintf("upperdir has unexpected value %q", o.Upper))
		}
	}

	if !o.ephemeral {
		if o.Upper != o.Work && (o.Upper == zeroString || o.Work == zeroString) {
			// unreachable
			return msg.WrapErr(ENOTRECOVERABLE, "impossible overlay state reached")
		}

		if o.Upper != zeroString {
			if !path.IsAbs(o.Upper) {
				return msg.WrapErr(EBADE, fmt.Sprintf("upperdir %q is not absolute", o.Upper))
			}
			if v, err := filepath.EvalSymlinks(o.Upper); err != nil {
				return wrapErrSelf(err)
			} else {
				o.Upper = escapeOverlayDataSegment(toHost(v))
			}
		}

		if o.Work != zeroString {
			if !path.IsAbs(o.Work) {
				return msg.WrapErr(EBADE, fmt.Sprintf("workdir %q is not absolute", o.Work))
			}
			if v, err := filepath.EvalSymlinks(o.Work); err != nil {
				return wrapErrSelf(err)
			} else {
				o.Work = escapeOverlayDataSegment(toHost(v))
			}
		}
	}

	for i := range o.Lower {
		if !path.IsAbs(o.Lower[i]) {
			return msg.WrapErr(EBADE, fmt.Sprintf("lowerdir %q is not absolute", o.Lower[i]))
		}

		if v, err := filepath.EvalSymlinks(o.Lower[i]); err != nil {
			return wrapErrSelf(err)
		} else {
			o.Lower[i] = escapeOverlayDataSegment(toHost(v))
		}
	}
	return nil
}

func (o *MountOverlayOp) apply(params *Params) error {
	if !path.IsAbs(o.Target) {
		return msg.WrapErr(EBADE, fmt.Sprintf("path %q is not absolute", o.Target))
	}
	target := toSysroot(o.Target)
	if err := os.MkdirAll(target, params.ParentPerm); err != nil {
		return wrapErrSelf(err)
	}

	if o.ephemeral {
		var err error
		// these directories are created internally, therefore early (absolute, symlink, prefix, escape) is bypassed
		if o.Upper, err = os.MkdirTemp(FHSRoot, intermediatePatternOverlayUpper); err != nil {
			return wrapErrSelf(err)
		}
		if o.Work, err = os.MkdirTemp(FHSRoot, intermediatePatternOverlayWork); err != nil {
			return wrapErrSelf(err)
		}
	}

	options := make([]string, 0, 4)

	if o.Upper == zeroString && o.Work == zeroString { // readonly
		if len(o.Lower) < 2 {
			return msg.WrapErr(EINVAL, "readonly overlay requires at least two lowerdir")
		}
		// "upperdir=" and "workdir=" may be omitted. In that case the overlay will be read-only
	} else {
		if len(o.Lower) == 0 {
			return msg.WrapErr(EINVAL, "overlay requires at least one lowerdir")
		}
		options = append(options,
			OptionOverlayUpperdir+"="+o.Upper,
			OptionOverlayWorkdir+"="+o.Work)
	}
	options = append(options,
		OptionOverlayLowerdir+"="+strings.Join(o.Lower, SpecialOverlayPath),
		OptionOverlayUserxattr)

	return wrapErrSuffix(Mount(SourceOverlay, target, FstypeOverlay, 0, strings.Join(options, SpecialOverlayOption)),
		fmt.Sprintf("cannot mount overlay on %q:", o.Target))
}

func (o *MountOverlayOp) Is(op Op) bool {
	vo, ok := op.(*MountOverlayOp)
	return ok &&
		o.Target == vo.Target &&
		slices.Equal(o.Lower, vo.Lower) &&
		o.Upper == vo.Upper &&
		o.Work == vo.Work
}
func (*MountOverlayOp) prefix() string { return "mounting" }
func (o *MountOverlayOp) String() string {
	return fmt.Sprintf("overlay on %q with %d layers", o.Target, len(o.Lower))
}

func init() { gob.Register(new(SymlinkOp)) }

// Link appends an [Op] that creates a symlink in the container filesystem.
func (f *Ops) Link(target, linkName string) *Ops {
	*f = append(*f, &SymlinkOp{target, linkName})
	return f
}

type SymlinkOp [2]string

func (l *SymlinkOp) early(*Params) error {
	if strings.HasPrefix(l[0], "*") {
		l[0] = l[0][1:]
		if !path.IsAbs(l[0]) {
			return msg.WrapErr(EBADE, fmt.Sprintf("path %q is not absolute", l[0]))
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
		return msg.WrapErr(EBADE, fmt.Sprintf("path %q is not absolute", l[1]))
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

func init() { gob.Register(new(MkdirOp)) }

// Mkdir appends an [Op] that creates a directory in the container filesystem.
func (f *Ops) Mkdir(dest string, perm os.FileMode) *Ops {
	*f = append(*f, &MkdirOp{dest, perm})
	return f
}

type MkdirOp struct {
	Path string
	Perm os.FileMode
}

func (m *MkdirOp) early(*Params) error { return nil }
func (m *MkdirOp) apply(*Params) error {
	if !path.IsAbs(m.Path) {
		return msg.WrapErr(EBADE, fmt.Sprintf("path %q is not absolute", m.Path))
	}

	if err := os.MkdirAll(toSysroot(m.Path), m.Perm); err != nil {
		return wrapErrSelf(err)
	}
	return nil
}

func (m *MkdirOp) Is(op Op) bool  { vm, ok := op.(*MkdirOp); return ok && m == vm }
func (*MkdirOp) prefix() string   { return "creating" }
func (m *MkdirOp) String() string { return fmt.Sprintf("directory %q perm %s", m.Path, m.Perm) }

func init() { gob.Register(new(TmpfileOp)) }

// Place appends an [Op] that places a file in container path [TmpfileOp.Path] containing [TmpfileOp.Data].
func (f *Ops) Place(name string, data []byte) *Ops { *f = append(*f, &TmpfileOp{name, data}); return f }

// PlaceP is like Place but writes the address of [TmpfileOp.Data] to the pointer dataP points to.
func (f *Ops) PlaceP(name string, dataP **[]byte) *Ops {
	t := &TmpfileOp{Path: name}
	*dataP = &t.Data

	*f = append(*f, t)
	return f
}

type TmpfileOp struct {
	Path string
	Data []byte
}

func (t *TmpfileOp) early(*Params) error { return nil }
func (t *TmpfileOp) apply(params *Params) error {
	if !path.IsAbs(t.Path) {
		return msg.WrapErr(EBADE, fmt.Sprintf("path %q is not absolute", t.Path))
	}

	var tmpPath string
	if f, err := os.CreateTemp(FHSRoot, intermediatePatternTmpfile); err != nil {
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
		MS_RDONLY|MS_NODEV,
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
