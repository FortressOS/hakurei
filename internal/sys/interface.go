// Package sys wraps OS interaction library functions.
package sys

import (
	"io/fs"
	"log"
	"os/user"
	"strconv"

	"hakurei.app/container"
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
	// LookPath provides exec.LookPath.
	LookPath(file string) (string, error)
	// MustExecutable provides [container.MustExecutable].
	MustExecutable() string
	// LookupGroup provides [user.LookupGroup].
	LookupGroup(name string) (*user.Group, error)
	// ReadDir provides [os.ReadDir].
	ReadDir(name string) ([]fs.DirEntry, error)
	// Stat provides [os.Stat].
	Stat(name string) (fs.FileInfo, error)
	// Open provides [os.Open].
	Open(name string) (fs.File, error)
	// EvalSymlinks provides filepath.EvalSymlinks.
	EvalSymlinks(path string) (string, error)
	// Exit provides [os.Exit].
	Exit(code int)

	Println(v ...any)
	Printf(format string, v ...any)

	// Paths returns a populated [hst.Paths] struct.
	Paths() hst.Paths
	// Uid invokes hsu and returns target uid.
	// Any errors returned by Uid is already wrapped [hlog.BaseError].
	Uid(identity int) (int, error)
}

// GetUserID obtains user id from hsu by querying uid of identity 0.
func GetUserID(os State) (int, error) {
	if uid, err := os.Uid(0); err != nil {
		return -1, err
	} else {
		return (uid / 10000) - 100, nil
	}
}

// CopyPaths is a generic implementation of [hst.Paths].
func CopyPaths(os State, v *hst.Paths, userid int) {
	if tempDir, err := container.NewAbs(os.TempDir()); err != nil {
		log.Fatalf("invalid TMPDIR: %v", err)
	} else {
		v.TempDir = tempDir
	}

	v.SharePath = v.TempDir.Append("hakurei." + strconv.Itoa(userid))
	hlog.Verbosef("process share directory at %q", v.SharePath)

	r, _ := os.LookupEnv(xdgRuntimeDir)
	if a, err := container.NewAbs(r); err != nil {
		// fall back to path in share since hakurei has no hard XDG dependency
		v.RunDirPath = v.SharePath.Append("run")
		v.RuntimePath = v.RunDirPath.Append("compat")
	} else {
		v.RuntimePath = a
		v.RunDirPath = v.RuntimePath.Append("hakurei")
	}
	hlog.Verbosef("runtime directory at %q", v.RunDirPath)
}
