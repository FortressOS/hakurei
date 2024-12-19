package linux

import (
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"sync"

	"git.gensokyo.uk/security/fortify/internal"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
)

// Std implements System using the standard library.
type Std struct {
	paths     Paths
	pathsOnce sync.Once

	uidOnce sync.Once
	uidCopy map[int]struct {
		uid int
		err error
	}
	uidMu sync.RWMutex
}

func (s *Std) Geteuid() int                                 { return os.Geteuid() }
func (s *Std) LookupEnv(key string) (string, bool)          { return os.LookupEnv(key) }
func (s *Std) TempDir() string                              { return os.TempDir() }
func (s *Std) LookPath(file string) (string, error)         { return exec.LookPath(file) }
func (s *Std) Executable() (string, error)                  { return os.Executable() }
func (s *Std) LookupGroup(name string) (*user.Group, error) { return user.LookupGroup(name) }
func (s *Std) ReadDir(name string) ([]os.DirEntry, error)   { return os.ReadDir(name) }
func (s *Std) Stat(name string) (fs.FileInfo, error)        { return os.Stat(name) }
func (s *Std) Open(name string) (fs.File, error)            { return os.Open(name) }
func (s *Std) Exit(code int)                                { fmsg.Exit(code) }
func (s *Std) Stdout() io.Writer                            { return os.Stdout }

const xdgRuntimeDir = "XDG_RUNTIME_DIR"

func (s *Std) Paths() Paths {
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

	s.uidMu.RLock()
	if u, ok := s.uidCopy[aid]; ok {
		s.uidMu.RUnlock()
		return u.uid, u.err
	}

	s.uidMu.RUnlock()
	s.uidMu.Lock()
	defer s.uidMu.Unlock()

	u := struct {
		uid int
		err error
	}{}
	defer func() { s.uidCopy[aid] = u }()

	u.uid = -1
	if fsu, ok := internal.Check(internal.Fsu); !ok {
		fmsg.Fatal("invalid fsu path, this copy of fshim is not compiled correctly")
		panic("unreachable")
	} else {
		cmd := exec.Command(fsu)
		cmd.Path = fsu
		cmd.Stderr = os.Stderr // pass through fatal messages
		cmd.Env = []string{"FORTIFY_APP_ID=" + strconv.Itoa(aid)}
		cmd.Dir = "/"
		var p []byte
		if p, u.err = cmd.Output(); u.err == nil {
			u.uid, u.err = strconv.Atoi(string(p))
		}
		return u.uid, u.err
	}
}
