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
		return msg.WrapErr(err, err.Error())
	}
	f, err := os.OpenFile(name, syscall.O_CREAT|syscall.O_EXCL|syscall.O_WRONLY, perm)
	if err != nil {
		return msg.WrapErr(err, err.Error())
	}
	if content != nil {
		_, err = f.Write(content)
		if err != nil {
			err = msg.WrapErr(err, err.Error())
		}
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
		err = msg.WrapErr(syscall.EISDIR,
			fmt.Sprintf("path %q is a directory", name))
	}
	return err
}

var hostProc = newProcPats(hostPath)

func newProcPats(prefix string) *procPaths {
	return &procPaths{prefix, prefix + "/self", prefix + "/self/mountinfo"}
}

type procPaths struct {
	prefix    string
	self      string
	mountinfo string
}

func (p *procPaths) stdout() string   { return p.self + "/fd/1" }
func (p *procPaths) fd(fd int) string { return p.self + "/fd/" + strconv.Itoa(fd) }
