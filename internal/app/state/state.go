// Package state provides cross-process state tracking for hakurei container instances.
package state

import (
	"errors"

	"hakurei.app/hst"
)

// ErrNoConfig is returned by [Cursor] when used with a nil [hst.Config].
var ErrNoConfig = errors.New("state does not contain config")

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

// Cursor provides access to the store of an identity.
type Cursor interface {
	Save(state *hst.State) error
	Destroy(id hst.ID) error
	Load() (map[hst.ID]*hst.State, error)
	Len() (int, error)
}
