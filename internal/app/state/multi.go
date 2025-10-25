package state

import (
	"encoding/gob"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strconv"
	"sync"

	"hakurei.app/hst"
	"hakurei.app/internal/lockedfile"
	"hakurei.app/message"
)

// multiLockFileName is the name of the file backing [lockedfile.Mutex] of a multiBackend.
const multiLockFileName = "lock"

// fine-grained locking and access
type multiStore struct {
	base string

	// initialised backends
	backends *sync.Map

	msg message.Msg
	mu  sync.RWMutex
}

func (s *multiStore) Do(identity int, f func(c Cursor)) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// load or initialise new backend
	b := new(multiBackend)
	b.mu.Lock()
	if v, ok := s.backends.LoadOrStore(identity, b); ok {
		b = v.(*multiBackend)
	} else {
		b.path = path.Join(s.base, strconv.Itoa(identity))

		// ensure directory
		if err := os.MkdirAll(b.path, 0700); err != nil && !errors.Is(err, fs.ErrExist) {
			s.backends.CompareAndDelete(identity, b)
			return false, &hst.AppError{Step: "create store segment directory", Err: err}
		}

		// set up file-based mutex
		b.lockfile = lockedfile.MutexAt(path.Join(b.path, multiLockFileName))

		b.mu.Unlock()
	}

	// lock backend
	if unlock, err := b.lockfile.Lock(); err != nil {
		return false, &hst.AppError{Step: "lock store segment", Err: err}
	} else {
		// unlock backend after Do is complete
		defer unlock()
	}

	// expose backend methods without exporting the pointer
	c := new(struct{ *multiBackend })
	c.multiBackend = b
	f(c)
	// disable access to the backend on a best-effort basis
	c.multiBackend = nil

	return true, nil
}

func (s *multiStore) List() ([]int, error) {
	var entries []os.DirEntry

	// read base directory to get all identities
	if v, err := os.ReadDir(s.base); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, &hst.AppError{Step: "read store directory", Err: err}
	} else {
		entries = v
	}

	aidsBuf := make([]int, 0, len(entries))
	for _, e := range entries {
		// skip non-directories
		if !e.IsDir() {
			s.msg.Verbosef("skipped non-directory entry %q", e.Name())
			continue
		}

		// skip non-numerical names
		if v, err := strconv.Atoi(e.Name()); err != nil {
			s.msg.Verbosef("skipped non-aid entry %q", e.Name())
			continue
		} else {
			if v < hst.IdentityMin || v > hst.IdentityMax {
				s.msg.Verbosef("skipped out of bounds entry %q", e.Name())
				continue
			}

			aidsBuf = append(aidsBuf, v)
		}
	}

	return append([]int(nil), aidsBuf...), nil
}

type multiBackend struct {
	path string

	// created/opened by prepare
	lockfile *lockedfile.Mutex

	mu sync.RWMutex
}

func (b *multiBackend) filename(id *hst.ID) string { return path.Join(b.path, id.String()) }

// reads all launchers in multiBackend
// file contents are ignored if decode is false
func (b *multiBackend) load(decode bool) (map[hst.ID]*hst.State, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// read directory contents, should only contain files named after ids
	var entries []os.DirEntry
	if pl, err := os.ReadDir(b.path); err != nil {
		return nil, &hst.AppError{Step: "read store segment directory", Err: err}
	} else {
		entries = pl
	}

	// allocate as if every entry is valid
	// since that should be the case assuming no external interference happens
	r := make(map[hst.ID]*hst.State, len(entries))

	for _, e := range entries {
		if e.IsDir() {
			return nil, fmt.Errorf("unexpected directory %q in store", e.Name())
		}

		// skip lock file
		if e.Name() == multiLockFileName {
			continue
		}

		var id hst.ID
		if err := id.UnmarshalText([]byte(e.Name())); err != nil {
			return nil, &hst.AppError{Step: "parse state key", Err: err}
		}

		// run in a function to better handle file closing
		if err := func() error {
			// open state file for reading
			if f, err := os.Open(path.Join(b.path, e.Name())); err != nil {
				return &hst.AppError{Step: "open state file", Err: err}
			} else {
				var s hst.State
				r[id] = &s

				// append regardless, but only parse if required, implements Len
				if decode {
					var et hst.Enablement
					if et, err = entryReadHeader(f); err != nil {
						_ = f.Close()
						return &hst.AppError{Step: "decode state header", Err: err}
					} else if err = gob.NewDecoder(f).Decode(&s); err != nil {
						_ = f.Close()
						return &hst.AppError{Step: "decode state body", Err: err}
					} else if s.ID != id {
						_ = f.Close()
						return fmt.Errorf("state entry %s has unexpected id %s", id, &s.ID)
					} else if err = f.Close(); err != nil {
						return &hst.AppError{Step: "close state file", Err: err}
					} else if err = s.Config.Validate(); err != nil {
						return err
					} else if s.Enablements.Unwrap() != et {
						return fmt.Errorf("state entry %s has unexpected enablement byte %x, %x", id, s.Enablements, et)
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

// Save writes process state to filesystem.
func (b *multiBackend) Save(state *hst.State) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if err := state.Config.Validate(); err != nil {
		return err
	}

	statePath := b.filename(&state.ID)
	if f, err := os.OpenFile(statePath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600); err != nil {
		return &hst.AppError{Step: "create state file", Err: err}
	} else if err = entryWriteHeader(f, state.Enablements.Unwrap()); err != nil {
		_ = f.Close()
		return &hst.AppError{Step: "encode state header", Err: err}
	} else if err = gob.NewEncoder(f).Encode(state); err != nil {
		_ = f.Close()
		return &hst.AppError{Step: "encode state body", Err: err}
	} else if err = f.Close(); err != nil {
		return &hst.AppError{Step: "close state file", Err: err}
	}
	return nil
}

func (b *multiBackend) Destroy(id hst.ID) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if err := os.Remove(b.filename(&id)); err != nil {
		return &hst.AppError{Step: "destroy state entry", Err: err}
	}
	return nil
}

func (b *multiBackend) Load() (map[hst.ID]*hst.State, error) { return b.load(true) }

func (b *multiBackend) Len() (int, error) {
	// rn consists of only nil entries but has the correct length
	rn, err := b.load(false)
	if err != nil {
		return -1, &hst.AppError{Step: "count state entries", Err: err}
	}
	return len(rn), nil
}

// NewMulti returns an instance of the multi-file store.
func NewMulti(msg message.Msg, runDir string) Store {
	return &multiStore{
		msg:      msg,
		base:     path.Join(runDir, "state"),
		backends: new(sync.Map),
	}
}
