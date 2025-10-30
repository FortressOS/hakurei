package store

import (
	"errors"
	"fmt"
	"iter"
	"os"
	"strconv"
	"sync"

	"hakurei.app/container/check"
	"hakurei.app/hst"
	"hakurei.app/internal/lockedfile"
)

// EntryHandle is a handle on a state entry retrieved from a [Handle].
// Must only be used while its parent [Handle.Lock] is held.
type EntryHandle struct {
	// Error returned while decoding pathname.
	// A non-nil value disables EntryHandle.
	DecodeErr error

	// Checked pathname to entry file.
	Pathname *check.Absolute

	hst.ID
}

// open opens the underlying state entry file.
// A non-nil error returned by open is of type [hst.AppError].
func (eh *EntryHandle) open(flag int, perm os.FileMode) (*os.File, error) {
	if eh.DecodeErr != nil {
		return nil, eh.DecodeErr
	}

	if f, err := os.OpenFile(eh.Pathname.String(), flag, perm); err != nil {
		return nil, &hst.AppError{Step: "open state entry", Err: err}
	} else {
		return f, nil
	}
}

// Destroy removes the underlying state entry.
// A non-nil error returned by Destroy is of type [hst.AppError].
func (eh *EntryHandle) Destroy() error {
	// destroy does not go through open
	if eh.DecodeErr != nil {
		return eh.DecodeErr
	}

	if err := os.Remove(eh.Pathname.String()); err != nil {
		return &hst.AppError{Step: "destroy state entry", Err: err}
	}
	return nil
}

// Save encodes [hst.State] and writes it to the underlying file.
// An error is returned if a file already exists with the same identifier.
// Save does not validate the embedded [hst.Config].
// A non-nil error returned by Save is of type [hst.AppError].
func (eh *EntryHandle) Save(state *hst.State) error {
	f, err := eh.open(os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}

	err = entryEncode(f, state)
	if closeErr := f.Close(); closeErr != nil && err == nil {
		err = &hst.AppError{Step: "close state file", Err: closeErr}
	}
	return err
}

// Load loads and validates the state entry header, and returns the [hst.Enablement] byte.
// for a non-nil v, the full state payload is decoded and stored in the value pointed to by v.
// Load validates the embedded [hst.Config] value.
// A non-nil error returned by Load is of type [hst.AppError].
func (eh *EntryHandle) Load(v *hst.State) (hst.Enablement, error) {
	f, err := eh.open(os.O_RDONLY, 0)
	if err != nil {
		return 0, err
	}

	var et hst.Enablement
	if v != nil {
		et, err = entryDecode(f, v)
		if err == nil && v.ID != eh.ID {
			err = &hst.AppError{Step: "validate state identifier", Err: os.ErrInvalid,
				Msg: fmt.Sprintf("state entry %s has unexpected id %s", eh.ID.String(), v.ID.String())}
		}
	} else {
		et, err = entryDecodeHeader(f)
	}

	if closeErr := f.Close(); closeErr != nil && err == nil {
		err = &hst.AppError{Step: "close state file", Err: closeErr}
	}
	return et, err
}

// Handle is a handle on a [Store] segment.
// Initialised by [Store.Handle].
type Handle struct {
	// Identity of instances tracked by this segment.
	Identity int
	// Pathname of directory that the segment referred to by Handle is rooted in.
	Path *check.Absolute

	// Inter-process mutex to synchronise operations against resources in this segment.
	// Must not be held directly, callers should use [Handle.Lock] instead.
	fileMu *lockedfile.Mutex
	// Must be held alongside fileMu.
	mu sync.Mutex
}

// Lock attempts to acquire a lock on [Handle].
// If successful, Lock returns a non-nil unlock function.
// A non-nil error returned by Lock is of type [hst.AppError].
func (h *Handle) Lock() (unlock func(), err error) {
	if unlock, err = h.fileMu.Lock(); err != nil {
		return nil, &hst.AppError{Step: "acquire lock on store segment " + strconv.Itoa(h.Identity), Err: err}
	}
	return
}

// Entries returns an iterator over all [EntryHandle] held in this segment.
// Must be called while holding [Handle.Lock].
// A non-nil error attached to a [EntryHandle] indicates a malformed identifier and is of type [hst.AppError].
// A non-nil error returned by Entries is of type [hst.AppError].
func (h *Handle) Entries() (iter.Seq[*EntryHandle], int, error) {
	// for error reporting
	const step = "read store segment entries"

	// read directory contents, should only contain storeMutexName and identifier
	var entries []os.DirEntry
	if pl, err := os.ReadDir(h.Path.String()); err != nil {
		return nil, -1, &hst.AppError{Step: step, Err: err}
	} else {
		entries = pl
	}

	// expects lock file
	l := len(entries)
	if l > 0 {
		l--
	}

	return func(yield func(*EntryHandle) bool) {
		for _, ent := range entries {
			var eh = EntryHandle{Pathname: h.Path.Append(ent.Name())}

			// this should never happen
			if ent.IsDir() {
				eh.DecodeErr = &hst.AppError{Step: step,
					Err: errors.New("unexpected directory " + strconv.Quote(ent.Name()) + " in store")}
				goto out
			}

			// silently skip lock file
			if ent.Name() == MutexName {
				continue
			}

			// this either indicates a serious bug or external interference
			if err := eh.ID.UnmarshalText([]byte(ent.Name())); err != nil {
				eh.DecodeErr = &hst.AppError{Step: "decode store segment entry", Err: err}
				goto out
			}

		out:
			if !yield(&eh) {
				break
			}
		}
	}, l, nil
}

// newHandle returns the address of a new segment [Handle] rooted in base.
func newHandle(base *check.Absolute, identity int) *Handle {
	h := Handle{Identity: identity, Path: base.Append(strconv.Itoa(identity))}
	h.fileMu = lockedfile.MutexAt(h.Path.Append(MutexName).String())
	return &h
}
