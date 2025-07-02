// Package sys wraps OS interaction library functions.
package sys

import (
	"io/fs"
	"os/user"
	"path"
	"strconv"

	"hakurei.app/hst"
	"hakurei.app/internal/hlog"
)

// State provides safe interaction with operating system state.
type State interface {
	// Getuid provides [os.Getuid].
	Getuid() int
	// Getgid provides [os.Getgid].
	Getgid() int
	// LookupEnv provides [os.LookupEnv].
	LookupEnv(key string) (string, bool)
	// TempDir provides [os.TempDir].
	TempDir() string
	// LookPath provides [exec.LookPath].
	LookPath(file string) (string, error)
	// MustExecutable provides [proc.MustExecutable].
	MustExecutable() string
	// LookupGroup provides [user.LookupGroup].
	LookupGroup(name string) (*user.Group, error)
	// ReadDir provides [os.ReadDir].
	ReadDir(name string) ([]fs.DirEntry, error)
	// Stat provides [os.Stat].
	Stat(name string) (fs.FileInfo, error)
	// Open provides [os.Open]
	Open(name string) (fs.File, error)
	// EvalSymlinks provides [filepath.EvalSymlinks]
	EvalSymlinks(path string) (string, error)
	// Exit provides [os.Exit].
	Exit(code int)

	Println(v ...any)
	Printf(format string, v ...any)

	// Paths returns a populated [hst.Paths] struct.
	Paths() hst.Paths
	// Uid invokes hsu and returns target uid.
	// Any errors returned by Uid is already wrapped [fmsg.BaseError].
	Uid(aid int) (int, error)
}

// CopyPaths is a generic implementation of [hst.Paths].
func CopyPaths(os State, v *hst.Paths) {
	v.SharePath = path.Join(os.TempDir(), "hakurei."+strconv.Itoa(os.Getuid()))

	hlog.Verbosef("process share directory at %q", v.SharePath)

	if r, ok := os.LookupEnv(xdgRuntimeDir); !ok || r == "" || !path.IsAbs(r) {
		// fall back to path in share since hakurei has no hard XDG dependency
		v.RunDirPath = path.Join(v.SharePath, "run")
		v.RuntimePath = path.Join(v.RunDirPath, "compat")
	} else {
		v.RuntimePath = r
		v.RunDirPath = path.Join(v.RuntimePath, "hakurei")
	}

	hlog.Verbosef("runtime directory at %q", v.RunDirPath)
}
