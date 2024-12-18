package state

import (
	"time"

	"git.ophivana.moe/security/fortify/fipc"
)

type Store interface {
	// Do calls f exactly once and ensures store exclusivity until f returns.
	// Returns whether f is called and any errors during the locking process.
	// Backend provided to f becomes invalid as soon as f returns.
	Do(f func(b Backend)) (bool, error)

	// Close releases any resources held by Store.
	Close() error
}

// Backend provides access to the store
type Backend interface {
	Save(state *State) error
	Destroy(pid int) error
	Load() ([]*State, error)
	Len() (int, error)
}

// State is the on-disk format for a fortified process's state information
type State struct {
	// fortify instance id
	ID [16]byte `json:"instance"`
	// child process PID value
	PID int `json:"pid"`
	// sealed app configuration
	Config *fipc.Config `json:"config"`

	// process start time
	Time time.Time
}
