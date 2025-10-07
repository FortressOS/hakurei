package container

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"

	"hakurei.app/container/fhs"
	"hakurei.app/container/vfs"
)

const (
	// Nonexistent is a path that cannot exist.
	// /proc is chosen because a system with covered /proc is unsupported by this package.
	Nonexistent = fhs.Proc + "nonexistent"

	hostPath    = fhs.Root + hostDir
	hostDir     = "host"
	sysrootPath = fhs.Root + sysrootDir
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

func ensureFile(name string, perm, pperm os.FileMode) error {
	fi, err := os.Stat(name)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		return createFile(name, perm, pperm, nil)
	}

	if mode := fi.Mode(); mode&fs.ModeDir != 0 || mode&fs.ModeSymlink != 0 {
		err = &os.PathError{Op: "ensure", Path: name, Err: syscall.EISDIR}
	}
	return err
}

var hostProc = newProcPaths(direct{}, hostPath)

func newProcPaths(k syscallDispatcher, prefix string) *procPaths {
	return &procPaths{k, prefix + "/proc", prefix + "/proc/self"}
}

type procPaths struct {
	k      syscallDispatcher
	prefix string
	self   string
}

func (p *procPaths) stdout() string   { return p.self + "/fd/1" }
func (p *procPaths) fd(fd int) string { return p.self + "/fd/" + strconv.Itoa(fd) }
func (p *procPaths) mountinfo(f func(d *vfs.MountInfoDecoder) error) error {
	if r, err := p.k.openNew(p.self + "/mountinfo"); err != nil {
		return err
	} else {
		d := vfs.NewMountInfoDecoder(r)
		err0 := f(d)
		if err = r.Close(); err != nil {
			return err
		} else if err = d.Err(); err != nil {
			return err
		}
		return err0
	}
}
