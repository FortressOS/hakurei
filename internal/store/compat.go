package store

import (
	"errors"
	"maps"
	"strconv"

	"hakurei.app/container/check"
	"hakurei.app/hst"
	"hakurei.app/message"
)

/* this provides an implementation of Store on top of the improved state tracking to ease in the changes */

type Compat interface {
	// Do calls f exactly once and ensures store exclusivity until f returns.
	// Returns whether f is called and any errors during the locking process.
	// Cursor provided to f becomes invalid as soon as f returns.
	Do(identity int, f func(c Cursor)) (ok bool, err error)

	// List queries the store and returns a list of identities known to the store.
	// Note that some or all returned identities might not have any active apps.
	List() (identities []int, err error)
}

func (s *stateStore) Do(identity int, f func(c Cursor)) (bool, error) {
	if h, err := s.identityHandle(identity); err != nil {
		return false, err
	} else {
		return h.do(f)
	}
}

// storeAdapter satisfies [Store] via stateStore.
type storeAdapter struct {
	msg message.Msg
	*stateStore
}

func (s storeAdapter) List() ([]int, error) {
	segments, n, err := s.segments()
	if err != nil {
		return nil, err
	}

	identities := make([]int, 0, n)
	for si := range segments {
		if si.err != nil {
			if m, ok := message.GetMessage(err); ok {
				s.msg.Verbose(m)
			} else {
				// unreachable
				return nil, err
			}
			continue
		}
		identities = append(identities, si.identity)
	}
	return identities, nil
}

// NewMulti returns an instance of the multi-file store.
func NewMulti(msg message.Msg, prefix *check.Absolute) Compat {
	return storeAdapter{msg, newStore(prefix.Append("state"))}
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

var (
	ErrDuplicate = errors.New("store contains duplicates")
)

// Joiner is the interface that wraps the Join method.
//
// The Join function uses Joiner if available.
type Joiner interface {
	Join() (map[hst.ID]*hst.State, error)
}

// Join returns joined state entries of all active identities.
func Join(s Compat) (map[hst.ID]*hst.State, error) {
	if j, ok := s.(Joiner); ok {
		return j.Join()
	}

	var (
		aids    []int
		entries = make(map[hst.ID]*hst.State)

		el      int
		res     map[hst.ID]*hst.State
		loadErr error
	)

	if ln, err := s.List(); err != nil {
		return nil, err
	} else {
		aids = ln
	}

	for _, aid := range aids {
		if _, err := s.Do(aid, func(c Cursor) {
			res, loadErr = c.Load()
		}); err != nil {
			return nil, err
		}

		if loadErr != nil {
			return nil, loadErr
		}

		// save expected length
		el = len(entries) + len(res)
		maps.Copy(entries, res)
		if len(entries) != el {
			return nil, ErrDuplicate
		}
	}

	return entries, nil
}
