package state

import (
	"errors"
	"io/fs"
	"iter"
	"os"
	"strconv"
	"sync"
	"syscall"

	"hakurei.app/container/check"
	"hakurei.app/hst"
	"hakurei.app/internal/lockedfile"
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
}

// bigLock acquires fileMu on stateStore.
// A non-nil error returned by bigLock is of type [hst.AppError].
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
// A non-nil error returned by identityHandle is of type [hst.AppError].
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

		err := os.MkdirAll(h.path.String(), 0700)
		h.mu.Unlock()
		if err != nil && !errors.Is(err, fs.ErrExist) {
			// handle methods will likely return ENOENT
			s.handles.CompareAndDelete(identity, h)
			return nil, &hst.AppError{Step: "create store segment directory", Err: err}
		}
	}
	return h, nil
}

// segmentIdentity is produced by the iterator returned by stateStore.segments.
type segmentIdentity struct {
	// Identity of the current segment.
	identity int
	// Error encountered while processing this segment.
	err error
}

// segments returns an iterator over all segmentIdentity known to the store.
// To obtain a storeHandle on a segment, caller must then call identityHandle.
// A non-nil error returned by segments is of type [hst.AppError].
func (s *stateStore) segments() (iter.Seq[segmentIdentity], int, error) {
	// read directory contents, should only contain storeMutexName and identity
	var entries []os.DirEntry

	// acquire big lock to read store segment list
	if unlock, err := s.bigLock(); err != nil {
		return nil, -1, err
	} else {
		entries, err = os.ReadDir(s.base.String())
		unlock()

		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, -1, &hst.AppError{Step: "read store segments", Err: err}
		}
	}

	// expects lock file
	l := len(entries)
	if l > 0 {
		l--
	}

	return func(yield func(segmentIdentity) bool) {
		// for error reporting
		const step = "process store segment"

		for _, ent := range entries {
			si := segmentIdentity{identity: -1}

			// should only be the big lock
			if !ent.IsDir() {
				if ent.Name() == storeMutexName {
					continue
				}

				// this should never happen
				si.err = &hst.AppError{Step: step, Err: syscall.EISDIR,
					Msg: "skipped non-directory entry " + strconv.Quote(ent.Name())}
				goto out
			}

			// failure paths either indicates a serious bug or external interference
			if v, err := strconv.Atoi(ent.Name()); err != nil {
				si.err = &hst.AppError{Step: step, Err: err,
					Msg: "skipped non-identity entry " + strconv.Quote(ent.Name())}
				goto out
			} else if v < hst.IdentityMin || v > hst.IdentityMax {
				si.err = &hst.AppError{Step: step, Err: syscall.ERANGE,
					Msg: "skipped out of bounds entry " + strconv.Itoa(v)}
				goto out
			} else {
				si.identity = v
			}

		out:
			if !yield(si) {
				break
			}
		}
	}, l, nil
}

// newStore returns the address of a new instance of stateStore.
// Multiple instances of stateStore rooted in the same directory is supported, but discouraged.
func newStore(base *check.Absolute) *stateStore {
	return &stateStore{base: base, fileMu: lockedfile.MutexAt(base.Append(storeMutexName).String())}
}
