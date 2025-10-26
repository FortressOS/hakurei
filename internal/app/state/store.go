package state

import (
	"errors"
	"io/fs"
	"os"
	"strconv"
	"sync"

	"hakurei.app/container/check"
	"hakurei.app/hst"
	"hakurei.app/internal/lockedfile"
	"hakurei.app/message"
)

// storeMutexName is the pathname of the file backing [lockedfile.Mutex] of a stateStore and storeHandle.
const storeMutexName = "lock"

// A stateStore keeps track of [hst.State] via a well-known filesystem accessible to all hakurei priv-side processes.
// Access to store data and related resources are synchronised on a per-segment basis via storeHandle.
type stateStore struct {
	// Pathname of directory that the store is rooted in.
	base *check.Absolute

	// All currently known instances of storeHandle, keyed by their identity.
	handles sync.Map

	// Inter-process mutex to synchronise operations against the entire store.
	// Held during List and when initialising previously unknown identities during Do.
	// Must not be accessed directly. Callers should use the bigLock method instead.
	fileMu *lockedfile.Mutex

	// For creating the base directory.
	mkdirOnce sync.Once
	// Stored error value via mkdirOnce.
	mkdirErr error

	msg message.Msg
}

// bigLock acquires fileMu on stateStore.
func (s *stateStore) bigLock() (unlock func(), err error) {
	s.mkdirOnce.Do(func() { s.mkdirErr = os.MkdirAll(s.base.String(), 0700) })
	if s.mkdirErr != nil {
		return nil, &hst.AppError{Step: "create state store directory", Err: s.mkdirErr}
	}

	if unlock, err = s.fileMu.Lock(); err != nil {
		return nil, &hst.AppError{Step: "acquire lock on the state store", Err: err}
	}
	return
}

// identityHandle loads or initialises a storeHandle for identity.
func (s *stateStore) identityHandle(identity int) (*storeHandle, error) {
	h := new(storeHandle)
	h.mu.Lock()

	if v, ok := s.handles.LoadOrStore(identity, h); ok {
		h = v.(*storeHandle)
	} else {
		// acquire big lock to initialise previously unknown segment handle
		if unlock, err := s.bigLock(); err != nil {
			return nil, err
		} else {
			defer unlock()
		}

		h.identity = identity
		h.path = s.base.Append(strconv.Itoa(identity))
		h.fileMu = lockedfile.MutexAt(h.path.Append(storeMutexName).String())

		if err := os.MkdirAll(h.path.String(), 0700); err != nil && !errors.Is(err, fs.ErrExist) {
			s.handles.CompareAndDelete(identity, h)
			return nil, &hst.AppError{Step: "create store segment directory", Err: err}
		}
		h.mu.Unlock()
	}
	return h, nil
}

func (s *stateStore) Do(identity int, f func(c Cursor)) (bool, error) {
	if h, err := s.identityHandle(identity); err != nil {
		return false, err
	} else {
		return h.do(f)
	}
}

func (s *stateStore) List() ([]int, error) {
	var entries []os.DirEntry

	// acquire big lock to read store segment list
	if unlock, err := s.bigLock(); err != nil {
		return nil, err
	} else {
		entries, err = os.ReadDir(s.base.String())
		unlock()

		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, &hst.AppError{Step: "read store directory", Err: err}
		}
	}

	identities := make([]int, 0, len(entries))
	for _, e := range entries {
		// should only be the big lock
		if !e.IsDir() {
			if e.Name() != storeMutexName {
				s.msg.Verbosef("skipped non-directory entry %q", e.Name())
			}
			continue
		}

		// this either indicates a serious bug or external interference
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
