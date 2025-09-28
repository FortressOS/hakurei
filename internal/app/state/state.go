// Package state provides cross-process state tracking for hakurei container instances.
package state

import (
	"errors"
	"io"
	"time"

	"hakurei.app/hst"
)

var ErrNoConfig = errors.New("state does not contain config")

type Entries map[ID]*State

type Store interface {
	// Do calls f exactly once and ensures store exclusivity until f returns.
	// Returns whether f is called and any errors during the locking process.
	// Cursor provided to f becomes invalid as soon as f returns.
	Do(identity int, f func(c Cursor)) (ok bool, err error)

	// List queries the store and returns a list of identities known to the store.
	// Note that some or all returned identities might not have any active apps.
	List() (identities []int, err error)

	// Close releases any resources held by Store.
	Close() error
}

// Cursor provides access to the store
type Cursor interface {
	Save(state *State, configWriter io.WriterTo) error
	Destroy(id ID) error
	Load() (Entries, error)
	Len() (int, error)
}

// State is an instance state
type State struct {
	// hakurei instance id
	ID ID `json:"instance"`
	// child process PID value
	PID int `json:"pid"`
	// sealed app configuration
	Config *hst.Config `json:"config"`

	// process start time
	Time time.Time `json:"time"`
}
