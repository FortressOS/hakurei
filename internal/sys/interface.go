package sys

import (
	"io/fs"
	"os/user"
	"path"
	"strconv"

	"git.gensokyo.uk/security/fortify/internal/app"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
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

	// Paths returns a populated [Paths] struct.
	Paths() app.Paths
	// Uid invokes fsu and returns target uid.
	// Any errors returned by Uid is already wrapped [fmsg.BaseError].
	Uid(aid int) (int, error)
}

// CopyPaths is a generic implementation of [fst.Paths].
func CopyPaths(os State, v *app.Paths) {
	v.SharePath = path.Join(os.TempDir(), "fortify."+strconv.Itoa(os.Getuid()))

	fmsg.Verbosef("process share directory at %q", v.SharePath)

	if r, ok := os.LookupEnv(xdgRuntimeDir); !ok || r == "" || !path.IsAbs(r) {
		// fall back to path in share since fortify has no hard XDG dependency
		v.RunDirPath = path.Join(v.SharePath, "run")
		v.RuntimePath = path.Join(v.RunDirPath, "compat")
	} else {
		v.RuntimePath = r
		v.RunDirPath = path.Join(v.RuntimePath, "fortify")
	}

	fmsg.Verbosef("runtime directory at %q", v.RunDirPath)
}
