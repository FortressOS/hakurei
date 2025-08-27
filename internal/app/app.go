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

func New(ctx context.Context, os sys.State) (*App, error) {
	a := new(App)
	a.sys = os
	a.ctx = ctx

	id := new(state.ID)
	err := state.NewAppID(id)
	a.id = newID(id)

	return a, err
}

func MustNew(ctx context.Context, os sys.State) *App {
	a, err := New(ctx, os)
	if err != nil {
		log.Fatalf("cannot create app: %v", err)
	}
	return a
}

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
		return "(invalid app)"
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.outcome != nil {
		if a.outcome.user.uid == nil {
			return fmt.Sprintf("(sealed app %s with invalid uid)", a.id)
		}
		return fmt.Sprintf("(sealed app %s as uid %s)", a.id, a.outcome.user.uid)
	}

	return fmt.Sprintf("(unsealed app %s)", a.id)
}

// Seal determines the outcome of [hst.Config] as a [SealedApp].
// Values stored in and referred to by [hst.Config] might be overwritten and must not be used again.
func (a *App) Seal(config *hst.Config) (*Outcome, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.outcome != nil {
		panic("app sealed twice")
	}

	seal := new(Outcome)
	seal.id = a.id
	err := seal.finalise(a.ctx, a.sys, config)
	if err == nil {
		a.outcome = seal
	}
	return seal, err
}
