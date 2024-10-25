package internal

import (
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path"
	"strconv"
	"sync"

	"git.ophivana.moe/security/fortify/internal/fmsg"
)

// System provides safe access to operating system resources.
type System interface {
	// Geteuid provides [os.Geteuid].
	Geteuid() int
	// LookupEnv provides [os.LookupEnv].
	LookupEnv(key string) (string, bool)
	// TempDir provides [os.TempDir].
	TempDir() string
	// LookPath provides [exec.LookPath].
	LookPath(file string) (string, error)
	// Executable provides [os.Executable].
	Executable() (string, error)
	// Lookup provides [user.Lookup].
	Lookup(username string) (*user.User, error)
	// ReadDir provides [os.ReadDir].
	ReadDir(name string) ([]fs.DirEntry, error)
	// Stat provides [os.Stat].
	Stat(name string) (fs.FileInfo, error)
	// Open provides [os.Open]
	Open(name string) (fs.File, error)
	// Exit provides [os.Exit].
	Exit(code int)

	// Paths returns a populated [Paths] struct.
	Paths() Paths
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
func CopyPaths(os System, v *Paths) {
	v.SharePath = path.Join(os.TempDir(), "fortify."+strconv.Itoa(os.Geteuid()))

	fmsg.VPrintf("process share directory at %q", v.SharePath)

	if r, ok := os.LookupEnv(xdgRuntimeDir); !ok || r == "" || !path.IsAbs(r) {
		// fall back to path in share since fortify has no hard XDG dependency
		v.RunDirPath = path.Join(v.SharePath, "run")
		v.RuntimePath = path.Join(v.RunDirPath, "compat")
	} else {
		v.RuntimePath = r
		v.RunDirPath = path.Join(v.RuntimePath, "fortify")
	}

	fmsg.VPrintf("runtime directory at %q", v.RunDirPath)
}

// Std implements System using the standard library.
type Std struct {
	paths     Paths
	pathsOnce sync.Once
}

func (s *Std) Geteuid() int {
	return os.Geteuid()
}

func (s *Std) LookupEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}

func (s *Std) TempDir() string {
	return os.TempDir()
}

func (s *Std) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (s *Std) Executable() (string, error) {
	return os.Executable()
}

func (s *Std) Lookup(username string) (*user.User, error) {
	return user.Lookup(username)
}

func (s *Std) ReadDir(name string) ([]os.DirEntry, error) {
	return os.ReadDir(name)
}

func (s *Std) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

func (s *Std) Open(name string) (fs.File, error) {
	return os.Open(name)
}
func (s *Std) Exit(code int) {
	os.Exit(code)
}

const xdgRuntimeDir = "XDG_RUNTIME_DIR"

func (s *Std) Paths() Paths {
	s.pathsOnce.Do(func() { CopyPaths(s, &s.paths) })
	return s.paths
}
