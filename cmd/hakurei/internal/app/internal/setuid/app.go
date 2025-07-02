package setuid

import (
	"context"
	"fmt"
	"sync"

	. "git.gensokyo.uk/security/hakurei/cmd/hakurei/internal/app"
	"git.gensokyo.uk/security/hakurei/hst"
	"git.gensokyo.uk/security/hakurei/internal/hlog"
	"git.gensokyo.uk/security/hakurei/internal/sys"
)

func New(ctx context.Context, os sys.State) (App, error) {
	a := new(app)
	a.sys = os
	a.ctx = ctx

	id := new(ID)
	err := NewAppID(id)
	a.id = newID(id)

	return a, err
}

type app struct {
	id  *stringPair[ID]
	sys sys.State
	ctx context.Context

	*outcome
	mu sync.RWMutex
}

func (a *app) ID() ID { a.mu.RLock(); defer a.mu.RUnlock(); return a.id.unwrap() }

func (a *app) String() string {
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

func (a *app) Seal(config *hst.Config) (SealedApp, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.outcome != nil {
		panic("app sealed twice")
	}
	if config == nil {
		return nil, hlog.WrapErr(ErrConfig,
			"attempted to seal app with nil config")
	}

	seal := new(outcome)
	seal.id = a.id
	err := seal.finalise(a.ctx, a.sys, config)
	if err == nil {
		a.outcome = seal
	}
	return seal, err
}
