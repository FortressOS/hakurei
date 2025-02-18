package sys

import (
	"io/fs"
	"os/user"
	"path"
	"strconv"

	"git.gensokyo.uk/security/fortify/internal/fmsg"
)

// State provides safe interaction with operating system state.
type State interface {
	// Geteuid provides [os.Geteuid].
	Geteuid() int
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
	Paths() Paths
	// Uid invokes fsu and returns target uid.
	// Any errors returned by Uid is already wrapped [fmsg.BaseError].
	Uid(aid int) (int, error)
}

// Paths contains environment dependent paths used by fortify.
type Paths struct {
	// path to shared directory e.g. /tmp/fortify.%d
	SharePath string `json:"share_path"`
	// XDG_RUNTIME_DIR value e.g. /run/user/%d
	RuntimePath string `json:"runtime_path"`
	// application runtime directory e.g. /run/user/%d/fortify
	RunDirPath string `json:"run_dir_path"`
}

// CopyPaths is a generic implementation of [System.Paths].
func CopyPaths(os State, v *Paths) {
	v.SharePath = path.Join(os.TempDir(), "fortify."+strconv.Itoa(os.Geteuid()))

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
