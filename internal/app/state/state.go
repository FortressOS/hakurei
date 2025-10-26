// Package state provides cross-process state tracking for hakurei container instances.
package state

import (
	"strconv"

	"hakurei.app/container/check"
	"hakurei.app/hst"
	"hakurei.app/internal/lockedfile"
	"hakurei.app/message"
)

/* this provides an implementation of Store on top of the improved state tracking to ease in the changes */

type Store interface {
	// Do calls f exactly once and ensures store exclusivity until f returns.
	// Returns whether f is called and any errors during the locking process.
	// Cursor provided to f becomes invalid as soon as f returns.
	Do(identity int, f func(c Cursor)) (ok bool, err error)

	// List queries the store and returns a list of identities known to the store.
	// Note that some or all returned identities might not have any active apps.
	List() (identities []int, err error)
}

// NewMulti returns an instance of the multi-file store.
func NewMulti(msg message.Msg, prefix *check.Absolute) Store {
	store := &stateStore{msg: msg, base: prefix.Append("state")}
	store.fileMu = lockedfile.MutexAt(store.base.Append(storeMutexName).String())
	return store
}

// Cursor provides access to the store of an identity.
type Cursor interface {
	Save(state *hst.State) error
	Destroy(id hst.ID) error
	Load() (map[hst.ID]*hst.State, error)
	Len() (int, error)
}

// do implements stateStore.Do on storeHandle.
func (h *storeHandle) do(f func(c Cursor)) (bool, error) {
	if unlock, err := h.fileMu.Lock(); err != nil {
		return false, &hst.AppError{Step: "acquire lock on store segment " + strconv.Itoa(h.identity), Err: err}
	} else {
		defer unlock()
	}

	f(h)
	return true, nil
}

/* these compatibility methods must only be called while fileMu is held */

func (h *storeHandle) Save(state *hst.State) error {
	return (&stateEntryHandle{nil, h.path.Append(state.ID.String()), state.ID}).save(state)
}

func (h *storeHandle) Destroy(id hst.ID) error {
	return (&stateEntryHandle{nil, h.path.Append(id.String()), id}).destroy()
}

func (h *storeHandle) Load() (map[hst.ID]*hst.State, error) {
	entries, n, err := h.entries()
	if err != nil {
		return nil, err
	}

	r := make(map[hst.ID]*hst.State, n)
	for eh := range entries {
		if eh.decodeErr != nil {
			err = eh.decodeErr
			break
		}
		var s hst.State
		if _, err = eh.load(&s); err != nil {
			break
		}
		r[eh.ID] = &s
	}
	return r, err
}

func (h *storeHandle) Len() (int, error) {
	entries, _, err := h.entries()
	if err != nil {
		return -1, err
	}

	var n int
	for eh := range entries {
		if eh.decodeErr != nil {
			err = eh.decodeErr
		}
		n++
	}
	return n, err
}
