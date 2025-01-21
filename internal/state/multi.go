package state

import (
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strconv"
	"sync"
	"syscall"

	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
)

// fine-grained locking and access
type multiStore struct {
	base string

	// initialised backends
	backends *sync.Map

	lock sync.RWMutex
}

func (s *multiStore) Do(aid int, f func(c Cursor)) (bool, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	// load or initialise new backend
	b := new(multiBackend)
	if v, ok := s.backends.LoadOrStore(aid, b); ok {
		b = v.(*multiBackend)
	} else {
		b.lock.Lock()
		b.path = path.Join(s.base, strconv.Itoa(aid))

		// ensure directory
		if err := os.MkdirAll(b.path, 0700); err != nil && !errors.Is(err, fs.ErrExist) {
			s.backends.CompareAndDelete(aid, b)
			return false, err
		}

		// open locker file
		if l, err := os.OpenFile(b.path+".lock", os.O_RDWR|os.O_CREATE, 0600); err != nil {
			s.backends.CompareAndDelete(aid, b)
			return false, err
		} else {
			b.lockfile = l
		}
		b.lock.Unlock()
	}

	// lock backend
	if err := b.lockFile(); err != nil {
		return false, err
	}

	// expose backend methods without exporting the pointer
	c := new(struct{ *multiBackend })
	c.multiBackend = b
	f(b)
	// disable access to the backend on a best-effort basis
	c.multiBackend = nil

	// unlock backend
	return true, b.unlockFile()
}

func (s *multiStore) List() ([]int, error) {
	var entries []os.DirEntry

	// read base directory to get all aids
	if v, err := os.ReadDir(s.base); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	} else {
		entries = v
	}

	aidsBuf := make([]int, 0, len(entries))
	for _, e := range entries {
		// skip non-directories
		if !e.IsDir() {
			fmsg.VPrintf("skipped non-directory entry %q", e.Name())
			continue
		}

		// skip non-numerical names
		if v, err := strconv.Atoi(e.Name()); err != nil {
			fmsg.VPrintf("skipped non-aid entry %q", e.Name())
			continue
		} else {
			if v < 0 || v > 9999 {
				fmsg.VPrintf("skipped out of bounds entry %q", e.Name())
				continue
			}

			aidsBuf = append(aidsBuf, v)
		}
	}

	return append([]int(nil), aidsBuf...), nil
}

func (s *multiStore) Close() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	var errs []error
	s.backends.Range(func(_, value any) bool {
		b := value.(*multiBackend)
		errs = append(errs, b.close())
		return true
	})

	return errors.Join(errs...)
}

type multiBackend struct {
	path string

	// created/opened by prepare
	lockfile *os.File

	lock sync.RWMutex
}

func (b *multiBackend) filename(id *fst.ID) string {
	return path.Join(b.path, id.String())
}

func (b *multiBackend) lockFileAct(lt int) (err error) {
	op := "LockAct"
	switch lt {
	case syscall.LOCK_EX:
		op = "Lock"
	case syscall.LOCK_UN:
		op = "Unlock"
	}

	for {
		err = syscall.Flock(int(b.lockfile.Fd()), lt)
		if !errors.Is(err, syscall.EINTR) {
			break
		}
	}
	if err != nil {
		return &fs.PathError{
			Op:   op,
			Path: b.lockfile.Name(),
			Err:  err,
		}
	}
	return nil
}

func (b *multiBackend) lockFile() error {
	return b.lockFileAct(syscall.LOCK_EX)
}

func (b *multiBackend) unlockFile() error {
	return b.lockFileAct(syscall.LOCK_UN)
}

// reads all launchers in simpleBackend
// file contents are ignored if decode is false
func (b *multiBackend) load(decode bool) (Entries, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	// read directory contents, should only contain files named after ids
	var entries []os.DirEntry
	if pl, err := os.ReadDir(b.path); err != nil {
		return nil, err
	} else {
		entries = pl
	}

	// allocate as if every entry is valid
	// since that should be the case assuming no external interference happens
	r := make(Entries, len(entries))

	for _, e := range entries {
		if e.IsDir() {
			return nil, fmt.Errorf("unexpected directory %q in store", e.Name())
		}

		id := new(fst.ID)
		if err := fst.ParseAppID(id, e.Name()); err != nil {
			return nil, err
		}

		// run in a function to better handle file closing
		if err := func() error {
			// open state file for reading
			if f, err := os.Open(path.Join(b.path, e.Name())); err != nil {
				return err
			} else {
				defer func() {
					if f.Close() != nil {
						// unreachable
						panic("foreign state file closed prematurely")
					}
				}()

				s := new(State)
				r[*id] = s

				// append regardless, but only parse if required, implements Len
				if decode {
					if err = b.decodeState(f, s); err != nil {
						return err
					}
					if s.ID != *id {
						return fmt.Errorf("state entry %s has unexpected id %s", id, &s.ID)
					}
				}

				return nil
			}
		}(); err != nil {
			return nil, err
		}
	}

	return r, nil
}

// state file consists of an eight byte header, followed by concatenated gobs
// of [fst.Config] and [State], if [State.Config] is not nil or offset < 0,
// the first gob is skipped
func (b *multiBackend) decodeState(r io.ReadSeeker, state *State) error {
	offset := make([]byte, 8)
	if l, err := r.Read(offset); err != nil {
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("state file too short: %d bytes", l)
		}
		return err
	}

	// decode volatile state first
	var skipConfig bool
	{
		o := int64(binary.LittleEndian.Uint64(offset))
		skipConfig = o < 0

		if !skipConfig {
			if l, err := r.Seek(o, io.SeekCurrent); err != nil {
				return err
			} else if l != 8+o {
				return fmt.Errorf("invalid seek offset %d", l)
			}
		}
	}
	if err := gob.NewDecoder(r).Decode(state); err != nil {
		return err
	}

	// decode sealed config
	if state.Config == nil {
		// config must be provided either as part of volatile state,
		// or in the config segment
		if skipConfig {
			return ErrNoConfig
		}

		state.Config = new(fst.Config)
		if _, err := r.Seek(8, io.SeekStart); err != nil {
			return err
		}
		return gob.NewDecoder(r).Decode(state.Config)
	} else {
		return nil
	}
}

// Save writes process state to filesystem
func (b *multiBackend) Save(state *State, configWriter io.WriterTo) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if configWriter == nil && state.Config == nil {
		return ErrNoConfig
	}

	statePath := b.filename(&state.ID)

	if f, err := os.OpenFile(statePath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600); err != nil {
		return err
	} else {
		defer func() {
			if f.Close() != nil {
				// unreachable
				panic("state file closed prematurely")
			}
		}()
		return b.encodeState(f, state, configWriter)
	}
}

func (b *multiBackend) encodeState(w io.WriteSeeker, state *State, configWriter io.WriterTo) error {
	offset := make([]byte, 8)

	// skip header bytes
	if _, err := w.Seek(8, io.SeekStart); err != nil {
		return err
	}

	if configWriter != nil {
		// write config gob and encode header
		if l, err := configWriter.WriteTo(w); err != nil {
			return err
		} else {
			binary.LittleEndian.PutUint64(offset, uint64(l))
		}
	} else {
		// offset == -1 indicates absence of config gob
		binary.LittleEndian.PutUint64(offset, 0xffffffffffffffff)
	}

	// encode volatile state
	if err := gob.NewEncoder(w).Encode(state); err != nil {
		return err
	}

	// write header
	if _, err := w.Seek(0, io.SeekStart); err != nil {
		return err
	}
	_, err := w.Write(offset)
	return err
}

func (b *multiBackend) Destroy(id fst.ID) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	return os.Remove(b.filename(&id))
}

func (b *multiBackend) Load() (Entries, error) {
	return b.load(true)
}

func (b *multiBackend) Len() (int, error) {
	// rn consists of only nil entries but has the correct length
	rn, err := b.load(false)
	return len(rn), err
}

func (b *multiBackend) close() error {
	b.lock.Lock()
	defer b.lock.Unlock()

	err := b.lockfile.Close()
	if err == nil || errors.Is(err, os.ErrInvalid) || errors.Is(err, os.ErrClosed) {
		return nil
	}
	return err
}

// NewMulti returns an instance of the multi-file store.
func NewMulti(runDir string) Store {
	b := new(multiStore)
	b.base = path.Join(runDir, "state")
	b.backends = new(sync.Map)
	return b
}
