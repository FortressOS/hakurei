package state

import (
	"errors"
	"io"
	"time"

	"git.gensokyo.uk/security/fortify/fst"
)

var ErrNoConfig = errors.New("state does not contain config")

type Entries map[fst.ID]*State

type Store interface {
	// Do calls f exactly once and ensures store exclusivity until f returns.
	// Returns whether f is called and any errors during the locking process.
	// Cursor provided to f becomes invalid as soon as f returns.
	Do(aid int, f func(c Cursor)) (ok bool, err error)

	// List queries the store and returns a list of aids known to the store.
	// Note that some or all returned aids might not have any active apps.
	List() (aids []int, err error)

	// Close releases any resources held by Store.
	Close() error
}

// Cursor provides access to the store
type Cursor interface {
	Save(state *State, configWriter io.WriterTo) error
	Destroy(id fst.ID) error
	Load() (Entries, error)
	Len() (int, error)
}

// State is a fortify process's state
type State struct {
	// fortify instance id
	ID fst.ID `json:"instance"`
	// child process PID value
	PID int `json:"pid"`
	// sealed app configuration
	Config *fst.Config `json:"config"`

	// process start time
	Time time.Time `json:"time"`
}
