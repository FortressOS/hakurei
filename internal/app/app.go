package app

import (
	"fmt"
	"sync"

	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/internal/app/shim"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
	"git.gensokyo.uk/security/fortify/internal/sys"
)

func New(os sys.State) (fst.App, error) {
	a := new(app)
	a.sys = os

	id := new(fst.ID)
	err := fst.NewAppID(id)
	a.id = newID(id)

	return a, err
}

type app struct {
	// application unique identifier
	id *stringPair[fst.ID]
	// operating system interface
	sys sys.State
	// shim process manager
	shim *shim.Shim

	mu sync.RWMutex
	*appSeal
}

func (a *app) ID() fst.ID { return a.id.unwrap() }

func (a *app) String() string {
	if a == nil {
		return "(invalid app)"
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.shim != nil {
		return a.shim.String()
	}

	if a.appSeal != nil {
		if a.appSeal.user.uid == nil {
			return fmt.Sprintf("(sealed app %s with invalid uid)", a.id)
		}
		return fmt.Sprintf("(sealed app %s as uid %s)", a.id, a.appSeal.user.uid)
	}

	return fmt.Sprintf("(unsealed app %s)", a.id)
}

func (a *app) Seal(config *fst.Config) (err error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.appSeal != nil {
		panic("app sealed twice")
	}
	if config == nil {
		return fmsg.WrapError(ErrConfig,
			"attempted to seal app with nil config")
	}

	seal := new(appSeal)
	err = seal.finalise(a.sys, config, a.id.String())
	if err == nil {
		a.appSeal = seal
	}
	return
}
