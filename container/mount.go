package container

import (
	"errors"
	"fmt"
	"os"
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
	// SourceOverlay is used when mounting overlay.
	// Note that any source value is allowed when fstype is [FstypeOverlay].
	SourceOverlay = "overlay"

	// SourceTmpfs is used when mounting tmpfs.
	SourceTmpfs = "tmpfs"
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
	// FstypeOverlay represents the overlay pseudo-filesystem.
	// This filesystem type can be mounted anywhere in the container filesystem.
	FstypeOverlay = "overlay"

	// OptionOverlayLowerdir represents the lowerdir option of the overlay pseudo-filesystem.
	// Any filesystem, does not need to be on a writable filesystem.
	OptionOverlayLowerdir = "lowerdir"
	// OptionOverlayUpperdir represents the upperdir option of the overlay pseudo-filesystem.
	// The upperdir is normally on a writable filesystem.
	OptionOverlayUpperdir = "upperdir"
	// OptionOverlayWorkdir represents the workdir option of the overlay pseudo-filesystem.
	// The workdir needs to be an empty directory on the same filesystem as upperdir.
	OptionOverlayWorkdir = "workdir"
	// OptionOverlayUserxattr represents the userxattr option of the overlay pseudo-filesystem.
	// Use the "user.overlay." xattr namespace instead of "trusted.overlay.".
	OptionOverlayUserxattr = "userxattr"

	// SpecialOverlayEscape is the escape string for overlay mount options.
	SpecialOverlayEscape = `\`
	// SpecialOverlayOption is the separator string between overlay mount options.
	SpecialOverlayOption = ","
	// SpecialOverlayPath is the separator string between overlay paths.
	SpecialOverlayPath = ":"
)

// bindMount mounts source on target and recursively applies flags if MS_REC is set.
func (p *procPaths) bindMount(msg Msg, source, target string, flags uintptr) error {
	// syscallDispatcher.bindMount and procPaths.remount must not be called from this function

	if err := p.k.mount(source, target, FstypeNULL, MS_SILENT|MS_BIND|flags&MS_REC, zeroString); err != nil {
		return err
	}
	return p.k.remount(msg, target, flags)
}

// remount applies flags on target, recursively if MS_REC is set.
func (p *procPaths) remount(msg Msg, target string, flags uintptr) error {
	// syscallDispatcher methods bindMount, remount must not be called from this function

	var targetFinal string
	if v, err := p.k.evalSymlinks(target); err != nil {
		return err
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
			destFd, err = p.k.open(targetFinal, O_PATH|O_CLOEXEC, 0)
			return
		}); err != nil {
			return &os.PathError{Op: "open", Path: targetFinal, Err: err}
		}
		if v, err := p.k.readlink(p.fd(destFd)); err != nil {
			return err
		} else if err = p.k.close(destFd); err != nil {
			return &os.PathError{Op: "close", Path: targetFinal, Err: err}
		} else {
			targetKFinal = v
		}
	}

	mf := MS_NOSUID | flags&MS_NODEV | flags&MS_RDONLY
	return p.mountinfo(func(d *vfs.MountInfoDecoder) error {
		n, err := d.Unfold(targetKFinal)
		if err != nil {
			return err
		}

		if err = remountWithFlags(p.k, msg, n, mf); err != nil {
			return err
		}
		if flags&MS_REC == 0 {
			return nil
		}

		for cur := range n.Collective() {
			// avoid remounting twice
			if cur == n {
				continue
			}

			if err = remountWithFlags(p.k, msg, cur, mf); err != nil && !errors.Is(err, EACCES) {
				return err
			}
		}

		return nil
	})
}

// remountWithFlags remounts mount point described by [vfs.MountInfoNode].
func remountWithFlags(k syscallDispatcher, msg Msg, n *vfs.MountInfoNode, mf uintptr) error {
	// syscallDispatcher methods bindMount, remount must not be called from this function

	kf, unmatched := n.Flags()
	if len(unmatched) != 0 {
		msg.Verbosef("unmatched vfs options: %q", unmatched)
	}

	if kf&mf != mf {
		return k.mount(SourceNone, n.Clean, FstypeNULL, MS_SILENT|MS_BIND|MS_REMOUNT|kf|mf, zeroString)
	}
	return nil
}

// mountTmpfs mounts tmpfs on target;
// callers who wish to mount to sysroot must pass the return value of toSysroot.
func mountTmpfs(k syscallDispatcher, fsname, target string, flags uintptr, size int, perm os.FileMode) error {
	// syscallDispatcher.mountTmpfs must not be called from this function

	if err := k.mkdirAll(target, parentPerm(perm)); err != nil {
		return err
	}
	opt := fmt.Sprintf("mode=%#o", perm)
	if size > 0 {
		opt += fmt.Sprintf(",size=%d", size)
	}
	return k.mount(fsname, target, FstypeTmpfs, flags, opt)
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

// EscapeOverlayDataSegment escapes a string for formatting into the data argument of an overlay mount call.
func EscapeOverlayDataSegment(s string) string {
	if s == zeroString {
		return zeroString
	}

	if f := strings.SplitN(s, "\x00", 2); len(f) > 0 {
		s = f[0]
	}

	return strings.NewReplacer(
		SpecialOverlayEscape, SpecialOverlayEscape+SpecialOverlayEscape,
		SpecialOverlayOption, SpecialOverlayEscape+SpecialOverlayOption,
		SpecialOverlayPath, SpecialOverlayEscape+SpecialOverlayPath,
	).Replace(s)
}
