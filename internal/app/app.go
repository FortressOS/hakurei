package app

import (
	"sync"

	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/internal/app/shim"
	"git.gensokyo.uk/security/fortify/internal/sys"
)

func New(os sys.State) (fst.App, error) {
	a := new(app)
	a.id = new(fst.ID)
	a.os = os
	return a, fst.NewAppID(a.id)
}

type app struct {
	// application unique identifier
	id *fst.ID
	// operating system interface
	os sys.State
	// shim process manager
	shim *shim.Shim
	// child process related information
	seal *appSeal

	lock sync.RWMutex
}

func (a *app) ID() fst.ID {
	return *a.id
}

func (a *app) String() string {
	if a == nil {
		return "(invalid fortified app)"
	}

	a.lock.RLock()
	defer a.lock.RUnlock()

	if a.shim != nil {
		return a.shim.String()
	}

	if a.seal != nil {
		return "(sealed fortified app as uid " + a.seal.sys.user.us + ")"
	}

	return "(unsealed fortified app)"
}
