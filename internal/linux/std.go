package linux

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"sync"

	"git.ophivana.moe/security/fortify/internal"
	"git.ophivana.moe/security/fortify/internal/fmsg"
)

// Std implements System using the standard library.
type Std struct {
	paths     Paths
	pathsOnce sync.Once

	sdBooted     bool
	sdBootedOnce sync.Once

	fshim     string
	fshimOnce sync.Once
}

func (s *Std) Geteuid() int                               { return os.Geteuid() }
func (s *Std) LookupEnv(key string) (string, bool)        { return os.LookupEnv(key) }
func (s *Std) TempDir() string                            { return os.TempDir() }
func (s *Std) LookPath(file string) (string, error)       { return exec.LookPath(file) }
func (s *Std) Executable() (string, error)                { return os.Executable() }
func (s *Std) Lookup(username string) (*user.User, error) { return user.Lookup(username) }
func (s *Std) ReadDir(name string) ([]os.DirEntry, error) { return os.ReadDir(name) }
func (s *Std) Stat(name string) (fs.FileInfo, error)      { return os.Stat(name) }
func (s *Std) Open(name string) (fs.File, error)          { return os.Open(name) }
func (s *Std) Exit(code int)                              { fmsg.Exit(code) }

const xdgRuntimeDir = "XDG_RUNTIME_DIR"

func (s *Std) FshimPath() string {
	s.fshimOnce.Do(func() {
		p, ok := internal.Path(internal.Fshim)
		if !ok {
			fmsg.Fatal("invalid fshim path, this copy of fortify is not compiled correctly")
		}
		s.fshim = p
	})

	return s.fshim
}

func (s *Std) Paths() Paths {
	s.pathsOnce.Do(func() { CopyPaths(s, &s.paths) })
	return s.paths
}

func (s *Std) SdBooted() bool {
	s.sdBootedOnce.Do(func() { s.sdBooted = copySdBooted() })
	return s.sdBooted
}

const systemdCheckPath = "/run/systemd/system"

func copySdBooted() bool {
	if v, err := sdBooted(); err != nil {
		fmsg.Println("cannot read systemd marker:", err)
		return false
	} else {
		return v
	}
}

func sdBooted() (bool, error) {
	_, err := os.Stat(systemdCheckPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			err = nil
		}
		return false, err
	}

	return true, nil
}
