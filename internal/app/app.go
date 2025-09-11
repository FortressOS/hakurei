// Package app implements high-level hakurei container behaviour.
package app

import (
	"context"
	"fmt"
	"log"
	"sync"

	"hakurei.app/hst"
	"hakurei.app/internal/app/state"
	"hakurei.app/internal/sys"
)

// New returns the address of a newly initialised [App] struct.
func New(ctx context.Context, os sys.State) (*App, error) {
	a := new(App)
	a.sys = os
	a.ctx = ctx

	id := new(state.ID)
	err := state.NewAppID(id)
	a.id = newID(id)

	return a, err
}

// MustNew calls [New] and panics if an error is returned.
func MustNew(ctx context.Context, os sys.State) *App {
	a, err := New(ctx, os)
	if err != nil {
		log.Fatalf("cannot create app: %v", err)
	}
	return a
}

// An App keeps track of the hakurei container lifecycle.
type App struct {
	outcome *Outcome

	id  *stringPair[state.ID]
	sys sys.State
	ctx context.Context
	mu  sync.RWMutex
}

// ID returns a copy of [state.ID] held by App.
func (a *App) ID() state.ID { a.mu.RLock(); defer a.mu.RUnlock(); return a.id.unwrap() }

func (a *App) String() string {
	if a == nil {
		return "<nil>"
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.outcome != nil {
		if a.outcome.user.uid == nil {
			return "<invalid>"
		}
		return fmt.Sprintf("sealed app %s as uid %s", a.id, a.outcome.user.uid)
	}

	return fmt.Sprintf("unsealed app %s", a.id)
}

// Seal determines the [Outcome] of [hst.Config].
// Values stored in and referred to by [hst.Config] might be overwritten and must not be used again.
func (a *App) Seal(config *hst.Config) (*Outcome, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.outcome != nil {
		panic("attempting to seal app twice")
	}

	seal := new(Outcome)
	seal.id = a.id
	err := seal.finalise(a.ctx, a.sys, config)
	if err == nil {
		a.outcome = seal
	}
	return seal, err
}
