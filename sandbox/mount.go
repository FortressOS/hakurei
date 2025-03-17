package sandbox

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
)

const (
	BindOptional = 1 << iota
	BindSource
	BindWritable
	BindDevice

	bindResolved
	bindAbsolute
	bindRecursive
)

func bindMount(src, dest string, flags int) error {
	target := toSysroot(dest)
	var source string

	if flags&BindSource != 0 {
		if flags&BindOptional != 0 {
			return msg.WrapErr(syscall.EINVAL,
				"flag source excludes optional")
		}
	} else if flags&bindResolved == 0 {
		return msg.WrapErr(syscall.EBADE,
			"flag source must be set on direct bind call")
	}

	if flags&bindAbsolute != 0 {
		if flags&BindSource == 0 {
			return msg.WrapErr(syscall.EINVAL,
				"flag absolute implies source")
		}
		source = src
	} else {
		source = toHost(src)
	}

	if fi, err := os.Stat(source); err != nil {
		return msg.WrapErr(err, err.Error())
	} else if fi.IsDir() {
		if err = os.MkdirAll(target, 0755); err != nil {
			return msg.WrapErr(err, err.Error())
		}
	} else if err = ensureFile(target, 0444); err != nil {
		if errors.Is(err, syscall.EISDIR) {
			return msg.WrapErr(err,
				fmt.Sprintf("path %q is a directory", dest))
		}
		return wrapErrSuffix(err,
			fmt.Sprintf("cannot create %q:", dest))
	}

	var mf uintptr = syscall.MS_SILENT | syscall.MS_BIND
	if flags&bindRecursive != 0 {
		mf |= syscall.MS_REC
	}
	if flags&BindWritable == 0 {
		mf |= syscall.MS_RDONLY
	}
	if flags&BindDevice == 0 {
		mf |= syscall.MS_NODEV
	}
	if msg.IsVerbose() {
		if strings.TrimPrefix(source, hostPath) == strings.TrimPrefix(target, sysrootPath) {
			msg.Verbosef("resolved %q flags %#x", target, mf)
		} else {
			msg.Verbosef("resolved %q on %q flags %#x", source, target, mf)
		}
	}
	return wrapErrSuffix(syscall.Mount(source, target, "", mf, ""),
		fmt.Sprintf("cannot bind %q on %q:", src, dest))
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
