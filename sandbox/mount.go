package sandbox

import (
	"fmt"
	"os"
	"syscall"
)

func (p *procPaths) bindMount(source, target string, flags uintptr, eq bool) error {
	var mf uintptr = syscall.MS_SILENT | syscall.MS_BIND
	mf |= flags & syscall.MS_REC
	if eq {
		msg.Verbosef("resolved %q flags %#x", target, mf)
	} else {
		msg.Verbosef("resolved %q on %q flags %#x", source, target, mf)
	}

	return wrapErrSuffix(syscall.Mount(source, target, "", mf, ""),
		fmt.Sprintf("cannot mount %q on %q:", source, target))
}

func mountTmpfs(fsname, name string, size int, perm os.FileMode) error {
	target := toSysroot(name)
	if err := os.MkdirAll(target, perm); err != nil {
		return msg.WrapErr(err, err.Error())
	}
	opt := fmt.Sprintf("mode=%#o", perm)
	if size > 0 {
		opt += fmt.Sprintf(",size=%d", size)
	}
	return wrapErrSuffix(syscall.Mount(fsname, target, "tmpfs",
		syscall.MS_NOSUID|syscall.MS_NODEV, opt),
		fmt.Sprintf("cannot mount tmpfs on %q:", name))
}
