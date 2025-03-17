package sandbox

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"strings"
	"syscall"
)

const (
	hostPath    = "/" + hostDir
	hostDir     = "host"
	sysrootPath = "/" + sysrootDir
	sysrootDir  = "sysroot"
)

func toSysroot(name string) string {
	name = strings.TrimLeftFunc(name, func(r rune) bool { return r == '/' })
	return path.Join(sysrootPath, name)
}

func toHost(name string) string {
	name = strings.TrimLeftFunc(name, func(r rune) bool { return r == '/' })
	return path.Join(hostPath, name)
}

func createFile(name string, perm os.FileMode, content []byte) error {
	if err := os.MkdirAll(path.Dir(name), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(name, syscall.O_CREAT|syscall.O_EXCL|syscall.O_WRONLY, perm)
	if err != nil {
		return err
	}
	if content != nil {
		_, err = f.Write(content)
	}
	return errors.Join(f.Close(), err)
}

func ensureFile(name string, perm os.FileMode) error {
	fi, err := os.Stat(name)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		return createFile(name, perm, nil)
	}

	if mode := fi.Mode(); mode&fs.ModeDir != 0 || mode&fs.ModeSymlink != 0 {
		err = syscall.EISDIR
	}
	return err
}
