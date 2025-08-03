package sys

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"

	"hakurei.app/container"
	"hakurei.app/hst"
	"hakurei.app/internal"
	"hakurei.app/internal/hlog"
)

// Std implements System using the standard library.
type Std struct {
	paths     hst.Paths
	pathsOnce sync.Once

	uidOnce sync.Once
	uidCopy map[int]struct {
		uid int
		err error
	}
	uidMu sync.RWMutex
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
	s.pathsOnce.Do(func() { CopyPaths(s, &s.paths) })
	return s.paths
}

func (s *Std) Uid(aid int) (int, error) {
	s.uidOnce.Do(func() {
		s.uidCopy = make(map[int]struct {
			uid int
			err error
		})
	})

	{
		s.uidMu.RLock()
		u, ok := s.uidCopy[aid]
		s.uidMu.RUnlock()
		if ok {
			return u.uid, u.err
		}
	}

	s.uidMu.Lock()
	defer s.uidMu.Unlock()

	u := struct {
		uid int
		err error
	}{}
	defer func() { s.uidCopy[aid] = u }()

	u.uid = -1
	hsuPath := internal.MustHsuPath()

	cmd := exec.Command(hsuPath)
	cmd.Path = hsuPath
	cmd.Stderr = os.Stderr // pass through fatal messages
	cmd.Env = []string{"HAKUREI_APP_ID=" + strconv.Itoa(aid)}
	cmd.Dir = container.FHSRoot
	var (
		p         []byte
		exitError *exec.ExitError
	)

	if p, u.err = cmd.Output(); u.err == nil {
		u.uid, u.err = strconv.Atoi(string(p))
		if u.err != nil {
			u.err = hlog.WrapErrSuffix(u.err, "cannot parse uid from hsu:")
		}
	} else if errors.As(u.err, &exitError) && exitError != nil && exitError.ExitCode() == 1 {
		u.err = hlog.WrapErr(syscall.EACCES, "") // hsu prints to stderr in this case
	} else if os.IsNotExist(u.err) {
		u.err = hlog.WrapErr(os.ErrNotExist, fmt.Sprintf("the setuid helper is missing: %s", hsuPath))
	}
	return u.uid, u.err
}
