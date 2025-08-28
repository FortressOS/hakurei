package container

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"

	"hakurei.app/container/vfs"
)

/* constants in this file bypass abs check, be extremely careful when changing them! */

const (
	// FHSRoot points to the file system root.
	FHSRoot = "/"
	// FHSEtc points to the directory for system-specific configuration.
	FHSEtc = "/etc/"
	// FHSTmp points to the place for small temporary files.
	FHSTmp = "/tmp/"

	// FHSRun points to a "tmpfs" file system for system packages to place runtime data, socket files, and similar.
	FHSRun = "/run/"
	// FHSRunUser points to a directory containing per-user runtime directories,
	// each usually individually mounted "tmpfs" instances.
	FHSRunUser = FHSRun + "user/"

	// FHSUsr points to vendor-supplied operating system resources.
	FHSUsr = "/usr/"
	// FHSUsrBin points to binaries and executables for user commands that shall appear in the $PATH search path.
	FHSUsrBin = FHSUsr + "bin/"

	// FHSVar points to persistent, variable system data. Writable during normal system operation.
	FHSVar = "/var/"
	// FHSVarLib points to persistent system data.
	FHSVarLib = FHSVar + "lib/"
	// FHSVarEmpty points to a nonstandard directory that is usually empty.
	FHSVarEmpty = FHSVar + "empty/"

	// FHSDev points to the root directory for device nodes.
	FHSDev = "/dev/"
	// FHSProc points to a virtual kernel file system exposing the process list and other functionality.
	FHSProc = "/proc/"
	// FHSProcSys points to a hierarchy below /proc/ that exposes a number of kernel tunables.
	FHSProcSys = FHSProc + "sys/"
	// FHSSys points to a virtual kernel file system exposing discovered devices and other functionality.
	FHSSys = "/sys/"
)

var (
	// AbsFHSRoot is [FHSRoot] as [Absolute].
	AbsFHSRoot = &Absolute{FHSRoot}
	// AbsFHSEtc is [FHSEtc] as [Absolute].
	AbsFHSEtc = &Absolute{FHSEtc}
	// AbsFHSTmp is [FHSTmp] as [Absolute].
	AbsFHSTmp = &Absolute{FHSTmp}

	// AbsFHSRun is [FHSRun] as [Absolute].
	AbsFHSRun = &Absolute{FHSRun}
	// AbsFHSRunUser is [FHSRunUser] as [Absolute].
	AbsFHSRunUser = &Absolute{FHSRunUser}

	// AbsFHSUsrBin is [FHSUsrBin] as [Absolute].
	AbsFHSUsrBin = &Absolute{FHSUsrBin}

	// AbsFHSVar is [FHSVar] as [Absolute].
	AbsFHSVar = &Absolute{FHSVar}
	// AbsFHSVarLib is [FHSVarLib] as [Absolute].
	AbsFHSVarLib = &Absolute{FHSVarLib}

	// AbsFHSDev is [FHSDev] as [Absolute].
	AbsFHSDev = &Absolute{FHSDev}
	// AbsFHSProc is [FHSProc] as [Absolute].
	AbsFHSProc = &Absolute{FHSProc}
	// AbsFHSSys is [FHSSys] as [Absolute].
	AbsFHSSys = &Absolute{FHSSys}
)

const (
	// Nonexistent is a path that cannot exist.
	// /proc is chosen because a system with covered /proc is unsupported by this package.
	Nonexistent = FHSProc + "nonexistent"

	hostPath    = FHSRoot + hostDir
	hostDir     = "host"
	sysrootPath = FHSRoot + sysrootDir
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
			return wrapErrSuffix(err,
				"cannot parse mountinfo:")
		}
		return err0
	}
}
