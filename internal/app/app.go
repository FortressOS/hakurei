package app

import (
	"fmt"
	"sync"

	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/internal/app/shim"
	"git.gensokyo.uk/security/fortify/internal/sys"
)

func New(os sys.State) (fst.App, error) {
	a := new(app)
	a.os = os

	id := new(fst.ID)
	err := fst.NewAppID(id)
	a.id = newID(id)

	return a, err
}

type app struct {
	// application unique identifier
	id *stringPair[fst.ID]
	// operating system interface
	os sys.State
	// shim process manager
	shim *shim.Shim
	// child process related information
	seal *appSeal

	lock sync.RWMutex
}

func (a *app) ID() fst.ID { return a.id.unwrap() }

func (a *app) String() string {
	if a == nil {
		return "(invalid app)"
	}

	a.lock.RLock()
	defer a.lock.RUnlock()

	if a.shim != nil {
		return a.shim.String()
	}

	if a.seal != nil {
		if a.seal.sys.user.uid == nil {
			return fmt.Sprintf("(sealed app %s with invalid uid)", a.id)
		}
		return fmt.Sprintf("(sealed app %s as uid %s)", a.id, a.seal.sys.user.uid)
	}

	return fmt.Sprintf("(unsealed app %s)", a.id)
}
