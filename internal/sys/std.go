package sys

import (
	"errors"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"sync"

	"hakurei.app/container"
	"hakurei.app/hst"
	"hakurei.app/internal"
	"hakurei.app/internal/hlog"
)

// Std implements System using the standard library.
type Std struct {
	paths     hst.Paths
	pathsOnce sync.Once
	Hsu
}

func (s *Std) Getuid() int                                  { return os.Getuid() }
func (s *Std) Getgid() int                                  { return os.Getgid() }
func (s *Std) LookupEnv(key string) (string, bool)          { return os.LookupEnv(key) }
func (s *Std) TempDir() string                              { return os.TempDir() }
func (s *Std) LookPath(file string) (string, error)         { return exec.LookPath(file) }
func (s *Std) MustExecutable() string                       { return container.MustExecutable() }
func (s *Std) LookupGroup(name string) (*user.Group, error) { return user.LookupGroup(name) }
func (s *Std) ReadDir(name string) ([]os.DirEntry, error)   { return os.ReadDir(name) }
func (s *Std) Stat(name string) (fs.FileInfo, error)        { return os.Stat(name) }
func (s *Std) Open(name string) (fs.File, error)            { return os.Open(name) }
func (s *Std) EvalSymlinks(path string) (string, error)     { return filepath.EvalSymlinks(path) }
func (s *Std) Exit(code int)                                { internal.Exit(code) }
func (s *Std) Println(v ...any)                             { hlog.Verbose(v...) }
func (s *Std) Printf(format string, v ...any)               { hlog.Verbosef(format, v...) }

const xdgRuntimeDir = "XDG_RUNTIME_DIR"

func (s *Std) Paths() hst.Paths {
	s.pathsOnce.Do(func() {
		if userid, err := GetUserID(s); err != nil {
			// TODO(ophestra): this duplicates code in cmd/hakurei/command.go, keep this up to date until removal
			hlog.BeforeExit()
			const fallback = "cannot obtain user id from hsu:"

			// this indicates the error message has already reached stderr, outside the current process's control;
			// this is only reached when hsu fails for any reason, as a second error message following hsu is confusing
			if errors.Is(err, ErrHsuAccess) {
				hlog.Verbose("*"+fallback, err)
				os.Exit(1)
				return
			}

			m, ok := container.GetErrorMessage(err)
			if !ok {
				log.Fatalln(fallback, err)
				return
			}

			log.Fatal(m)
		} else {
			CopyPaths(s, &s.paths, userid)
		}
	})
	return s.paths
}
