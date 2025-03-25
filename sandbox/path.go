package sandbox

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"

	"git.gensokyo.uk/security/fortify/sandbox/vfs"
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

func createFile(name string, perm, pperm os.FileMode, content []byte) error {
	if err := os.MkdirAll(path.Dir(name), pperm); err != nil {
		return wrapErrSelf(err)
	}
	f, err := os.OpenFile(name, syscall.O_CREAT|syscall.O_EXCL|syscall.O_WRONLY, perm)
	if err != nil {
		return wrapErrSelf(err)
	}
	if content != nil {
		_, err = f.Write(content)
		if err != nil {
			err = wrapErrSelf(err)
		}
	}
	return errors.Join(f.Close(), err)
}

func ensureFile(name string, perm, pperm os.FileMode) error {
	fi, err := os.Stat(name)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		return createFile(name, perm, pperm, nil)
	}

	if mode := fi.Mode(); mode&fs.ModeDir != 0 || mode&fs.ModeSymlink != 0 {
		err = msg.WrapErr(syscall.EISDIR,
			fmt.Sprintf("path %q is a directory", name))
	}
	return err
}

var hostProc = newProcPats(hostPath)

func newProcPats(prefix string) *procPaths {
	return &procPaths{prefix + "/proc", prefix + "/proc/self"}
}

type procPaths struct {
	prefix string
	self   string
}

func (p *procPaths) stdout() string   { return p.self + "/fd/1" }
func (p *procPaths) fd(fd int) string { return p.self + "/fd/" + strconv.Itoa(fd) }
func (p *procPaths) mountinfo(f func(d *vfs.MountInfoDecoder) error) error {
	if r, err := os.Open(p.self + "/mountinfo"); err != nil {
		return wrapErrSelf(err)
	} else {
		d := vfs.NewMountInfoDecoder(r)
		err0 := f(d)
		if err = r.Close(); err != nil {
			return wrapErrSuffix(err,
				"cannot close mountinfo:")
		} else if err = d.Err(); err != nil {
			return wrapErrSuffix(err,
				"cannot parse mountinfo:")
		}
		return err0
	}
}
