// Package store implements cross-process state tracking for hakurei container instances.
package store

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

// MutexName is the pathname of the file backing [lockedfile.Mutex] of a [Store] and [Handle].
const MutexName = "lock"

// A Store keeps track of [hst.State] via a well-known filesystem accessible to all hakurei priv-side processes.
// Access to store data and related resources are synchronised on a per-segment basis via [Handle].
type Store struct {
	// Pathname of directory that the store is rooted in.
	base *check.Absolute

	// All currently known instances of Handle, keyed by their identity.
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

// bigLock acquires fileMu on [Store].
// A non-nil error returned by bigLock is of type [hst.AppError].
func (s *Store) bigLock() (unlock func(), err error) {
	s.mkdirOnce.Do(func() { s.mkdirErr = os.MkdirAll(s.base.String(), 0700) })
	if s.mkdirErr != nil {
		return nil, &hst.AppError{Step: "create state store directory", Err: s.mkdirErr}
	}

	if unlock, err = s.fileMu.Lock(); err != nil {
		return nil, &hst.AppError{Step: "acquire lock on the state store", Err: err}
	}
	return
}

// Handle loads or initialises a [Handle] for identity.
// A non-nil error returned by Handle is of type [hst.AppError].
func (s *Store) Handle(identity int) (*Handle, error) {
	h := newHandle(s.base, identity)
	h.mu.Lock()

	if v, ok := s.handles.LoadOrStore(identity, h); ok {
		h = v.(*Handle)
	} else {
		// acquire big lock to initialise previously unknown segment handle
		if unlock, err := s.bigLock(); err != nil {
			return nil, err
		} else {
			defer unlock()
		}

		err := os.MkdirAll(h.Path.String(), 0700)
		h.mu.Unlock()
		if err != nil && !errors.Is(err, fs.ErrExist) {
			// handle methods will likely return ENOENT
			s.handles.CompareAndDelete(identity, h)
			return nil, &hst.AppError{Step: "create store segment directory", Err: err}
		}
	}
	return h, nil
}

// SegmentIdentity is produced by the iterator returned by [Store.Segments].
type SegmentIdentity struct {
	// Identity of the current segment.
	Identity int
	// Error encountered while processing this segment.
	Err error
}

// Segments returns an iterator over all [SegmentIdentity] known to the [Store].
// To obtain a [Handle] on a segment, caller must then call [Store.Handle].
// A non-nil error returned by segments is of type [hst.AppError].
func (s *Store) Segments() (iter.Seq[SegmentIdentity], int, error) {
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

	return func(yield func(SegmentIdentity) bool) {
		// for error reporting
		const step = "process store segment"

		for _, ent := range entries {
			si := SegmentIdentity{Identity: -1}

			// should only be the big lock
			if !ent.IsDir() {
				if ent.Name() == MutexName {
					continue
				}

				// this should never happen
				si.Err = &hst.AppError{Step: step, Err: syscall.ENOTDIR,
					Msg: "skipped non-directory entry " + strconv.Quote(ent.Name())}
				goto out
			}

			// failure paths either indicates a serious bug or external interference
			if v, err := strconv.Atoi(ent.Name()); err != nil {
				si.Err = &hst.AppError{Step: step, Err: err,
					Msg: "skipped non-identity entry " + strconv.Quote(ent.Name())}
				goto out
			} else if v < hst.IdentityMin || v > hst.IdentityMax {
				si.Err = &hst.AppError{Step: step, Err: syscall.ERANGE,
					Msg: "skipped out of bounds entry " + strconv.Itoa(v)}
				goto out
			} else {
				si.Identity = v
			}

		out:
			if !yield(si) {
				break
			}
		}
	}, l, nil
}

// All returns a non-reusable iterator over all [EntryHandle] known to this [Store].
// Callers must call copyError after completing iteration and handle the error accordingly.
// A non-nil error returned by copyError is of type [hst.AppError].
func (s *Store) All() (entries iter.Seq[*EntryHandle], copyError func() error) {
	var savedErr error
	return func(yield func(*EntryHandle) bool) {
		var segments iter.Seq[SegmentIdentity]
		segments, _, savedErr = s.Segments()
		if savedErr != nil {
			return
		}

		for si := range segments {
			if savedErr = si.Err; savedErr != nil {
				return
			}

			var handle *Handle
			if handle, savedErr = s.Handle(si.Identity); savedErr != nil {
				return // not reached
			}

			var unlock func()
			if unlock, savedErr = handle.Lock(); savedErr != nil {
				return
			}

			var segmentEntries iter.Seq[*EntryHandle]
			if segmentEntries, _, savedErr = handle.Entries(); savedErr != nil {
				unlock()
				return // not reached: lock has succeeded
			}

			for eh := range segmentEntries {
				if !yield(eh) {
					unlock()
					return
				}
			}
			unlock()
		}
	}, func() error { return savedErr }
}

// New returns the address of a new instance of [Store].
// Multiple instances of [Store] rooted in the same directory is possible, but unsupported.
func New(base *check.Absolute) *Store {
	return &Store{base: base, fileMu: lockedfile.MutexAt(base.Append(MutexName).String())}
}
