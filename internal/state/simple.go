package state

import (
	"encoding/gob"
	"errors"
	"io/fs"
	"os"
	"path"
	"strconv"
	"sync"
	"syscall"
)

// file-based locking
type simpleStore struct {
	path []string

	// created/opened by prepare
	lockfile *os.File
	// enforce prepare method
	init sync.Once
	// error returned by prepare
	initErr error

	lock sync.Mutex
}

func (s *simpleStore) Do(f func(b Backend)) (bool, error) {
	s.init.Do(s.prepare)
	if s.initErr != nil {
		return false, s.initErr
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	// lock store
	if err := s.lockFile(); err != nil {
		return false, err
	}

	// initialise new backend for caller
	b := new(simpleBackend)
	b.path = path.Join(s.path...)
	f(b)
	// disable backend
	b.lock.Lock()

	// unlock store
	return true, s.unlockFile()
}

func (s *simpleStore) lockFileAct(lt int) (err error) {
	op := "LockAct"
	switch lt {
	case syscall.LOCK_EX:
		op = "Lock"
	case syscall.LOCK_UN:
		op = "Unlock"
	}

	for {
		err = syscall.Flock(int(s.lockfile.Fd()), lt)
		if !errors.Is(err, syscall.EINTR) {
			break
		}
	}
	if err != nil {
		return &fs.PathError{
			Op:   op,
			Path: s.lockfile.Name(),
			Err:  err,
		}
	}
	return nil
}

func (s *simpleStore) lockFile() error {
	return s.lockFileAct(syscall.LOCK_EX)
}

func (s *simpleStore) unlockFile() error {
	return s.lockFileAct(syscall.LOCK_UN)
}

func (s *simpleStore) prepare() {
	s.initErr = func() error {
		prefix := path.Join(s.path...)
		// ensure directory
		if err := os.MkdirAll(prefix, 0700); err != nil && !errors.Is(err, fs.ErrExist) {
			return err
		}

		// open locker file
		if f, err := os.OpenFile(prefix+".lock", os.O_RDWR|os.O_CREATE, 0600); err != nil {
			return err
		} else {
			s.lockfile = f
		}

		return nil
	}()
}

func (s *simpleStore) Close() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	err := s.lockfile.Close()
	if err == nil || errors.Is(err, os.ErrInvalid) || errors.Is(err, os.ErrClosed) {
		return nil
	}
	return err
}

type simpleBackend struct {
	path string

	lock sync.RWMutex
}

func (b *simpleBackend) filename(pid int) string {
	return path.Join(b.path, strconv.Itoa(pid))
}

// reads all launchers in simpleBackend
// file contents are ignored if decode is false
func (b *simpleBackend) load(decode bool) ([]*State, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	var (
		r []*State
		f *os.File
	)

	// read directory contents, should only contain files named after PIDs
	if pl, err := os.ReadDir(b.path); err != nil {
		return nil, err
	} else {
		for _, e := range pl {
			// run in a function to better handle file closing
			if err = func() error {
				// open state file for reading
				if f, err = os.Open(path.Join(b.path, e.Name())); err != nil {
					return err
				} else {
					defer func() {
						if f.Close() != nil {
							// unreachable
							panic("foreign state file closed prematurely")
						}
					}()

					var s State
					r = append(r, &s)

					// append regardless, but only parse if required, used to implement Len
					if decode {
						return gob.NewDecoder(f).Decode(&s)
					} else {
						return nil
					}
				}
			}(); err != nil {
				return nil, err
			}
		}
	}

	return r, nil
}

// Save writes process state to filesystem
func (b *simpleBackend) Save(state *State) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if state.Config == nil {
		return errors.New("state does not contain config")
	}

	statePath := b.filename(state.PID)

	// create and open state data file
	if f, err := os.OpenFile(statePath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600); err != nil {
		return err
	} else {
		defer func() {
			if f.Close() != nil {
				// unreachable
				panic("state file closed prematurely")
			}
		}()
		// encode into state file
		return gob.NewEncoder(f).Encode(state)
	}
}

func (b *simpleBackend) Destroy(pid int) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	return os.Remove(b.filename(pid))
}

func (b *simpleBackend) Load() ([]*State, error) {
	return b.load(true)
}

func (b *simpleBackend) Len() (int, error) {
	// rn consists of only nil entries but has the correct length
	rn, err := b.load(false)
	return len(rn), err
}

// NewSimple returns an instance of a file-based store.
func NewSimple(runDir string, prefix ...string) Store {
	b := new(simpleStore)
	b.path = append([]string{runDir, "state"}, prefix...)
	return b
}
