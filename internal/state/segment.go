package state

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

// stateEntryHandle is a handle on a state entry retrieved from a storeHandle.
// Must only be used while its parent storeHandle.fileMu is held.
type stateEntryHandle struct {
	// Error returned while decoding pathname.
	// A non-nil value disables stateEntryHandle.
	decodeErr error

	// Checked path to entry file.
	pathname *check.Absolute

	hst.ID
}

// open opens the underlying state entry file, returning [hst.AppError] for a non-nil error.
func (eh *stateEntryHandle) open(flag int, perm os.FileMode) (*os.File, error) {
	if eh.decodeErr != nil {
		return nil, eh.decodeErr
	}

	if f, err := os.OpenFile(eh.pathname.String(), flag, perm); err != nil {
		return nil, &hst.AppError{Step: "open state entry", Err: err}
	} else {
		return f, nil
	}
}

// destroy removes the underlying state entry file, returning [hst.AppError] for a non-nil error.
func (eh *stateEntryHandle) destroy() error {
	// destroy does not go through open
	if eh.decodeErr != nil {
		return eh.decodeErr
	}

	if err := os.Remove(eh.pathname.String()); err != nil {
		return &hst.AppError{Step: "destroy state entry", Err: err}
	}
	return nil
}

// save encodes [hst.State] and writes it to the underlying file.
// An error is returned if a file already exists with the same identifier.
// save does not validate the embedded [hst.Config].
func (eh *stateEntryHandle) save(state *hst.State) error {
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

// load loads and validates the state entry header, and returns the [hst.Enablement] byte.
// for a non-nil v, the full state payload is decoded and stored in the value pointed to by v.
// load validates the embedded hst.Config value.
func (eh *stateEntryHandle) load(v *hst.State) (hst.Enablement, error) {
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

// storeHandle is a handle on a stateStore segment.
// Initialised by stateStore.identityHandle.
type storeHandle struct {
	// Identity of instances tracked by this segment.
	identity int
	// Pathname of directory that the segment referred to by storeHandle is rooted in.
	path *check.Absolute
	// Inter-process mutex to synchronise operations against resources in this segment.
	fileMu *lockedfile.Mutex

	// Must be held alongside fileMu.
	mu sync.Mutex
}

// entries returns an iterator over all stateEntryHandle held in this segment.
// Must be called while holding a lock on mu and fileMu.
// A non-nil error attached to a stateEntryHandle indicates a malformed identifier and is of type [hst.AppError].
// A non-nil error returned by entries is of type [hst.AppError].
func (h *storeHandle) entries() (iter.Seq[*stateEntryHandle], int, error) {
	// for error reporting
	const step = "read store segment entries"

	// read directory contents, should only contain storeMutexName and identifier
	var entries []os.DirEntry
	if pl, err := os.ReadDir(h.path.String()); err != nil {
		return nil, -1, &hst.AppError{Step: step, Err: err}
	} else {
		entries = pl
	}

	// expects lock file
	l := len(entries)
	if l > 0 {
		l--
	}

	return func(yield func(*stateEntryHandle) bool) {
		for _, ent := range entries {
			var eh = stateEntryHandle{pathname: h.path.Append(ent.Name())}

			// this should never happen
			if ent.IsDir() {
				eh.decodeErr = &hst.AppError{Step: step,
					Err: errors.New("unexpected directory " + strconv.Quote(ent.Name()) + " in store")}
				goto out
			}

			// silently skip lock file
			if ent.Name() == storeMutexName {
				continue
			}

			// this either indicates a serious bug or external interference
			if err := eh.ID.UnmarshalText([]byte(ent.Name())); err != nil {
				eh.decodeErr = &hst.AppError{Step: "decode store segment entry", Err: err}
				goto out
			}

		out:
			if !yield(&eh) {
				break
			}
		}
	}, l, nil
}
