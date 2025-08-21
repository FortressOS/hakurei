package container

import (
	"encoding/gob"
	"fmt"
	"path"
	. "syscall"
)

func init() { gob.Register(new(MountDevOp)) }

// Dev appends an [Op] that mounts a subset of host /dev.
func (f *Ops) Dev(target *Absolute, mqueue bool) *Ops {
	*f = append(*f, &MountDevOp{target, mqueue, false})
	return f
}

// DevWritable appends an [Op] that mounts a writable subset of host /dev.
// There is usually no good reason to write to /dev, so this should always be followed by a [RemountOp].
func (f *Ops) DevWritable(target *Absolute, mqueue bool) *Ops {
	*f = append(*f, &MountDevOp{target, mqueue, true})
	return f
}

// MountDevOp mounts a subset of host /dev on container path Target.
// If Mqueue is true, a private instance of [FstypeMqueue] is mounted.
// If Write is true, the resulting mount point is left writable.
type MountDevOp struct {
	Target *Absolute
	Mqueue bool
	Write  bool
}

func (d *MountDevOp) Valid() bool                                { return d != nil && d.Target != nil }
func (d *MountDevOp) early(*setupState, syscallDispatcher) error { return nil }
func (d *MountDevOp) apply(state *setupState, k syscallDispatcher) error {
	target := toSysroot(d.Target.String())

	if err := k.mountTmpfs(SourceTmpfsDevtmpfs, target, MS_NOSUID|MS_NODEV, 0, state.ParentPerm); err != nil {
		return err
	}

	for _, name := range []string{"null", "zero", "full", "random", "urandom", "tty"} {
		targetPath := path.Join(target, name)
		if err := k.ensureFile(targetPath, 0444, state.ParentPerm); err != nil {
			return err
		}
		if err := k.bindMount(
			toHost(FHSDev+name),
			targetPath,
			0,
			true,
		); err != nil {
			return err
		}
	}
	for i, name := range []string{"stdin", "stdout", "stderr"} {
		if err := k.symlink(
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
		if err := k.symlink(pair[0], path.Join(target, pair[1])); err != nil {
			return wrapErrSelf(err)
		}
	}

	devPtsPath := path.Join(target, "pts")
	for _, name := range []string{path.Join(target, "shm"), devPtsPath} {
		if err := k.mkdir(name, state.ParentPerm); err != nil {
			return wrapErrSelf(err)
		}
	}

	if err := k.mount(SourceDevpts, devPtsPath, FstypeDevpts, MS_NOSUID|MS_NOEXEC,
		"newinstance,ptmxmode=0666,mode=620"); err != nil {
		return wrapErrSuffix(err,
			fmt.Sprintf("cannot mount devpts on %q:", devPtsPath))
	}

	if state.RetainSession {
		if k.isatty(Stdout) {
			consolePath := path.Join(target, "console")
			if err := k.ensureFile(consolePath, 0444, state.ParentPerm); err != nil {
				return err
			}
			if name, err := k.readlink(hostProc.stdout()); err != nil {
				return wrapErrSelf(err)
			} else if err = k.bindMount(
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
		if err := k.mkdir(mqueueTarget, state.ParentPerm); err != nil {
			return wrapErrSelf(err)
		}
		if err := k.mount(SourceMqueue, mqueueTarget, FstypeMqueue, MS_NOSUID|MS_NOEXEC|MS_NODEV, zeroString); err != nil {
			return wrapErrSuffix(err, "cannot mount mqueue:")
		}
	}

	if d.Write {
		return nil
	}
	return wrapErrSuffix(k.remount(target, MS_RDONLY),
		fmt.Sprintf("cannot remount %q:", target))
}

func (d *MountDevOp) Is(op Op) bool {
	vd, ok := op.(*MountDevOp)
	return ok && d.Valid() && vd.Valid() &&
		d.Target.Is(vd.Target) &&
		d.Mqueue == vd.Mqueue &&
		d.Write == vd.Write
}
func (*MountDevOp) prefix() string { return "mounting" }
func (d *MountDevOp) String() string {
	if d.Mqueue {
		return fmt.Sprintf("dev on %q with mqueue", d.Target)
	}
	return fmt.Sprintf("dev on %q", d.Target)
}
