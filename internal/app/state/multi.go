package state

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"sync"

	"hakurei.app/container/check"
	"hakurei.app/hst"
	"hakurei.app/internal/lockedfile"
	"hakurei.app/message"
)

// multiLockFileName is the name of the file backing [lockedfile.Mutex] of a multiStore and multiBackend.
const multiLockFileName = "lock"

// fine-grained locking and access
type multiStore struct {
	// Pathname of directory that the store is rooted in.
	base *check.Absolute

	// All currently known instances of multiHandle, keyed by their identity.
	handles sync.Map
	// Held during List and when initialising previously unknown identities during Do.
	// Must not be accessed directly. Callers should use the bigLock method instead.
	fileMu *lockedfile.Mutex

	// For creating the base directory.
	mkdirOnce sync.Once
	// Stored error value via mkdirOnce.
	mkdirErr error

	msg message.Msg
	mu  sync.RWMutex
}

// bigLock acquires fileMu on multiStore.
// Must be called while holding a read lock on multiStore.
func (s *multiStore) bigLock() (unlock func(), err error) {
	s.mkdirOnce.Do(func() { s.mkdirErr = os.MkdirAll(s.base.String(), 0700) })
	if s.mkdirErr != nil {
		return nil, &hst.AppError{Step: "create state store directory", Err: s.mkdirErr}
	}

	if unlock, err = s.fileMu.Lock(); err != nil {
		return nil, &hst.AppError{Step: "acquire lock on the state store", Err: err}
	}
	return
}

// identityHandle loads or initialises a multiHandle for identity.
// Must be called while holding a read lock on multiStore.
func (s *multiStore) identityHandle(identity int) (*multiHandle, error) {
	b := new(multiHandle)
	b.mu.Lock()

	if v, ok := s.handles.LoadOrStore(identity, b); ok {
		b = v.(*multiHandle)
	} else {
		// acquire big lock to initialise previously unknown segment handle
		if unlock, err := s.bigLock(); err != nil {
			return nil, err
		} else {
			defer unlock()
		}

		b.path = s.base.Append(strconv.Itoa(identity))
		b.fileMu = lockedfile.MutexAt(b.path.Append(multiLockFileName).String())

		if err := os.MkdirAll(b.path.String(), 0700); err != nil && !errors.Is(err, fs.ErrExist) {
			s.handles.CompareAndDelete(identity, b)
			return nil, &hst.AppError{Step: "create store segment directory", Err: err}
		}
		b.mu.Unlock()
	}
	return b, nil
}

// do implements multiStore.Do on multiHandle.
func (h *multiHandle) do(identity int, f func(c Cursor)) (bool, error) {
	if unlock, err := h.fileMu.Lock(); err != nil {
		return false, &hst.AppError{Step: "acquire lock on store segment " + strconv.Itoa(identity), Err: err}
	} else {
		// unlock backend after Do is complete
		defer unlock()
	}

	// expose backend methods without exporting the pointer
	c := &struct{ *multiHandle }{h}
	f(c)
	// disable access to the backend on a best-effort basis
	c.multiHandle = nil
	return true, nil
}

func (s *multiStore) Do(identity int, f func(c Cursor)) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if h, err := s.identityHandle(identity); err != nil {
		return false, err
	} else {
		return h.do(identity, f)
	}
}

func (s *multiStore) List() ([]int, error) {
	var entries []os.DirEntry

	// acquire big lock to read store segment list
	s.mu.RLock()
	if unlock, err := s.bigLock(); err != nil {
		return nil, err
	} else {
		entries, err = os.ReadDir(s.base.String())
		s.mu.RUnlock()
		unlock()

		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, &hst.AppError{Step: "read store directory", Err: err}
		}
	}

	identities := make([]int, 0, len(entries))
	for _, e := range entries {
		// skip non-directories
		if !e.IsDir() {
			s.msg.Verbosef("skipped non-directory entry %q", e.Name())
			continue
		}

		// skip lock file
		if e.Name() == multiLockFileName {
			continue
		}

		// skip non-numerical names
		if v, err := strconv.Atoi(e.Name()); err != nil {
			s.msg.Verbosef("skipped non-identity entry %q", e.Name())
			continue
		} else {
			if v < hst.IdentityMin || v > hst.IdentityMax {
				s.msg.Verbosef("skipped out of bounds entry %q", e.Name())
				continue
			}

			identities = append(identities, v)
		}
	}

	return identities, nil
}

// multiHandle is a handle on a multiStore segment.
type multiHandle struct {
	// Pathname of directory that the segment referred to by multiHandle is rooted in.
	path *check.Absolute

	// created by prepare
	fileMu *lockedfile.Mutex

	mu sync.RWMutex
}

// instance returns the absolute pathname of a state entry file.
func (h *multiHandle) instance(id *hst.ID) *check.Absolute { return h.path.Append(id.String()) }

// load iterates over all [hst.State] entries reachable via multiHandle,
// decoding their contents if decode is true.
func (h *multiHandle) load(decode bool) (map[hst.ID]*hst.State, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// read directory contents, should only contain files named after ids
	var entries []os.DirEntry
	if pl, err := os.ReadDir(h.path.String()); err != nil {
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
			if f, err := os.Open(h.path.Append(e.Name()).String()); err != nil {
				return &hst.AppError{Step: "open state file", Err: err}
			} else {
				var s hst.State
				r[id] = &s

				// append regardless, but only parse if required, implements Len
				if decode {
					if err = entryDecode(f, &s); err != nil {
						_ = f.Close()
						return err
					} else if s.ID != id {
						return &hst.AppError{Step: "validate state identifier", Err: os.ErrInvalid,
							Msg: fmt.Sprintf("state entry %s has unexpected id %s", id, &s.ID)}
					}
				}

				if err = f.Close(); err != nil {
					return &hst.AppError{Step: "close state file", Err: err}
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
func (h *multiHandle) Save(state *hst.State) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if err := state.Config.Validate(); err != nil {
		return err
	}

	if f, err := os.OpenFile(h.instance(&state.ID).String(), os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600); err != nil {
		return &hst.AppError{Step: "create state file", Err: err}
	} else if err = entryEncode(f, state); err != nil {
		_ = f.Close()
		return err
	} else if err = f.Close(); err != nil {
		return &hst.AppError{Step: "close state file", Err: err}
	}
	return nil
}

func (h *multiHandle) Destroy(id hst.ID) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if err := os.Remove(h.instance(&id).String()); err != nil {
		return &hst.AppError{Step: "destroy state entry", Err: err}
	}
	return nil
}

func (h *multiHandle) Load() (map[hst.ID]*hst.State, error) { return h.load(true) }

func (h *multiHandle) Len() (int, error) {
	// rn consists of only nil entries but has the correct length
	rn, err := h.load(false)
	if err != nil {
		return -1, &hst.AppError{Step: "count state entries", Err: err}
	}
	return len(rn), nil
}

// NewMulti returns an instance of the multi-file store.
func NewMulti(msg message.Msg, prefix *check.Absolute) Store {
	store := &multiStore{msg: msg, base: prefix.Append("state")}
	store.fileMu = lockedfile.MutexAt(store.base.Append(multiLockFileName).String())
	return store
}
