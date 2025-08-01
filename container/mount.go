package container

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	. "syscall"

	"hakurei.app/container/vfs"
)

// bindMount mounts source on target and recursively applies flags if MS_REC is set.
func (p *procPaths) bindMount(source, target string, flags uintptr, eq bool) error {
	if eq {
		msg.Verbosef("resolved %q flags %#x", target, flags)
	} else {
		msg.Verbosef("resolved %q on %q flags %#x", source, target, flags)
	}

	if err := Mount(source, target, "", MS_SILENT|MS_BIND|flags&MS_REC, ""); err != nil {
		return wrapErrSuffix(err,
			fmt.Sprintf("cannot mount %q on %q:", source, target))
	}

	return p.remount(target, flags)
}

// remount applies flags on target, recursively if MS_REC is set.
func (p *procPaths) remount(target string, flags uintptr) error {
	var targetFinal string
	if v, err := filepath.EvalSymlinks(target); err != nil {
		return wrapErrSelf(err)
	} else {
		targetFinal = v
		if targetFinal != target {
			msg.Verbosef("target resolves to %q", targetFinal)
		}
	}

	// final target path according to the kernel through proc
	var targetKFinal string
	{
		var destFd int
		if err := IgnoringEINTR(func() (err error) {
			destFd, err = Open(targetFinal, O_PATH|O_CLOEXEC, 0)
			return
		}); err != nil {
			return wrapErrSuffix(err,
				fmt.Sprintf("cannot open %q:", targetFinal))
		}
		if v, err := os.Readlink(p.fd(destFd)); err != nil {
			return wrapErrSelf(err)
		} else if err = Close(destFd); err != nil {
			return wrapErrSuffix(err,
				fmt.Sprintf("cannot close %q:", targetFinal))
		} else {
			targetKFinal = v
		}
	}

	mf := MS_NOSUID | flags&MS_NODEV | flags&MS_RDONLY
	return hostProc.mountinfo(func(d *vfs.MountInfoDecoder) error {
		n, err := d.Unfold(targetKFinal)
		if err != nil {
			if errors.Is(err, ESTALE) {
				return msg.WrapErr(err,
					fmt.Sprintf("mount point %q never appeared in mountinfo", targetKFinal))
			}
			return wrapErrSuffix(err,
				"cannot unfold mount hierarchy:")
		}

		if err = remountWithFlags(n, mf); err != nil {
			return err
		}
		if flags&MS_REC == 0 {
			return nil
		}

		for cur := range n.Collective() {
			err = remountWithFlags(cur, mf)
			if err != nil && !errors.Is(err, EACCES) {
				return err
			}
		}

		return nil
	})
}

// remountWithFlags remounts mount point described by [vfs.MountInfoNode].
func remountWithFlags(n *vfs.MountInfoNode, mf uintptr) error {
	kf, unmatched := n.Flags()
	if len(unmatched) != 0 {
		msg.Verbosef("unmatched vfs options: %q", unmatched)
	}

	if kf&mf != mf {
		return wrapErrSuffix(
			Mount("none", n.Clean, "", MS_SILENT|MS_BIND|MS_REMOUNT|kf|mf, ""),
			fmt.Sprintf("cannot remount %q:", n.Clean))
	}
	return nil
}

func mountTmpfs(fsname, name string, flags uintptr, size int, perm os.FileMode) error {
	target := toSysroot(name)
	if err := os.MkdirAll(target, parentPerm(perm)); err != nil {
		return wrapErrSelf(err)
	}
	opt := fmt.Sprintf("mode=%#o", perm)
	if size > 0 {
		opt += fmt.Sprintf(",size=%d", size)
	}
	return wrapErrSuffix(
		Mount(fsname, target, "tmpfs", flags, opt),
		fmt.Sprintf("cannot mount tmpfs on %q:", name))
}

func parentPerm(perm os.FileMode) os.FileMode {
	pperm := 0755
	if perm&0070 == 0 {
		pperm &= ^0050
	}
	if perm&0007 == 0 {
		pperm &= ^0005
	}
	return os.FileMode(pperm)
}
