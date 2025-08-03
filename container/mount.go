package container

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	. "syscall"

	"hakurei.app/container/vfs"
)

/*
Holding CAP_SYS_ADMIN within the user namespace that owns a process's mount namespace
allows that process to create bind mounts and mount the following types of filesystems:
- /proc (since Linux 3.8)
- /sys (since Linux 3.8)
- devpts (since Linux 3.9)
- tmpfs(5) (since Linux 3.9)
- ramfs (since Linux 3.9)
- mqueue (since Linux 3.9)
- bpf (since Linux 4.4)
- overlayfs (since Linux 5.11)
*/

const (
	// zeroString is a zero value string, it represents NULL when passed to mount.
	zeroString = ""

	// SourceNone is used when the source value is ignored,
	// such as when remounting.
	SourceNone = "none"
	// SourceProc is used when mounting proc.
	// Note that any source value is allowed when fstype is [FstypeProc].
	SourceProc = "proc"
	// SourceDevpts is used when mounting devpts.
	// Note that any source value is allowed when fstype is [FstypeDevpts].
	SourceDevpts = "devpts"
	// SourceMqueue is used when mounting mqueue.
	// Note that any source value is allowed when fstype is [FstypeMqueue].
	SourceMqueue = "mqueue"

	// SourceTmpfsRootfs is used when mounting the tmpfs instance backing the intermediate root.
	SourceTmpfsRootfs = "rootfs"
	// SourceTmpfsDevtmpfs is used when mounting tmpfs representing a subset of host devtmpfs.
	SourceTmpfsDevtmpfs = "devtmpfs"
	// SourceTmpfsEphemeral is used when mounting a writable instance of tmpfs.
	SourceTmpfsEphemeral = "ephemeral"
	// SourceTmpfsReadonly is used when mounting a readonly instance of tmpfs.
	SourceTmpfsReadonly = "readonly"

	// FstypeNULL is used when the fstype value is ignored,
	// such as when bind mounting or remounting.
	FstypeNULL = zeroString
	// FstypeProc represents the proc pseudo-filesystem.
	// A fully visible instance of proc must be available in the mount namespace for proc to be mounted.
	// This filesystem type is usually mounted on [FHSProc].
	FstypeProc = "proc"
	// FstypeDevpts represents the devpts pseudo-filesystem.
	// This type of filesystem is usually mounted on /dev/pts.
	FstypeDevpts = "devpts"
	// FstypeTmpfs represents the tmpfs filesystem.
	// This filesystem type can be mounted anywhere in the container filesystem.
	FstypeTmpfs = "tmpfs"
	// FstypeMqueue represents the mqueue pseudo-filesystem.
	// This filesystem type is usually mounted on /dev/mqueue.
	FstypeMqueue = "mqueue"
)

// bindMount mounts source on target and recursively applies flags if MS_REC is set.
func (p *procPaths) bindMount(source, target string, flags uintptr, eq bool) error {
	if eq {
		msg.Verbosef("resolved %q flags %#x", target, flags)
	} else {
		msg.Verbosef("resolved %q on %q flags %#x", source, target, flags)
	}

	if err := Mount(source, target, FstypeNULL, MS_SILENT|MS_BIND|flags&MS_REC, zeroString); err != nil {
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
			Mount(SourceNone, n.Clean, FstypeNULL, MS_SILENT|MS_BIND|MS_REMOUNT|kf|mf, zeroString),
			fmt.Sprintf("cannot remount %q:", n.Clean))
	}
	return nil
}

// mountTmpfs mounts tmpfs on target;
// callers who wish to mount to sysroot must pass the return value of toSysroot.
func mountTmpfs(fsname, target string, flags uintptr, size int, perm os.FileMode) error {
	if err := os.MkdirAll(target, parentPerm(perm)); err != nil {
		return wrapErrSelf(err)
	}
	opt := fmt.Sprintf("mode=%#o", perm)
	if size > 0 {
		opt += fmt.Sprintf(",size=%d", size)
	}
	return wrapErrSuffix(
		Mount(fsname, target, FstypeTmpfs, flags, opt),
		fmt.Sprintf("cannot mount tmpfs on %q:", target))
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

// escapeOverlayDataSegment escapes a string for formatting into the data argument of an overlay mount call.
func escapeOverlayDataSegment(s string) string {
	if s == zeroString {
		return zeroString
	}

	if f := strings.SplitN(s, "\x00", 2); len(f) > 0 {
		s = f[0]
	}

	return strings.NewReplacer(
		`\`, `\\`,
		`,`, `\,`,
		`:`, `\:`,
	).Replace(s)
}
