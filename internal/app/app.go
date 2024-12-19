package app

import (
	"sync"
	"sync/atomic"

	"git.gensokyo.uk/security/fortify/cmd/fshim/ipc/shim"
	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/internal/linux"
)

type App interface {
	// ID returns a copy of App's unique ID.
	ID() fst.ID
	// Start sets up the system and starts the App.
	Start() error
	// Wait waits for App's process to exit and reverts system setup.
	Wait() (int, error)
	// WaitErr returns error returned by the underlying wait syscall.
	WaitErr() error

	Seal(config *fst.Config) error
	String() string
}

type app struct {
	// single-use config reference
	ct *appCt

	// application unique identifier
	id *fst.ID
	// operating system interface
	os linux.System
	// shim process manager
	shim *shim.Shim
	// child process related information
	seal *appSeal
	// error returned waiting for process
	waitErr error

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

func (a *app) WaitErr() error {
	return a.waitErr
}

func New(os linux.System) (App, error) {
	a := new(app)
	a.id = new(fst.ID)
	a.os = os
	return a, fst.NewAppID(a.id)
}

// appCt ensures its wrapped val is only accessed once
type appCt struct {
	val  *fst.Config
	done *atomic.Bool
}

func (a *appCt) Unwrap() *fst.Config {
	if !a.done.Load() {
		defer a.done.Store(true)
		return a.val
	}
	panic("attempted to access config reference twice")
}

func newAppCt(config *fst.Config) (ct *appCt) {
	ct = new(appCt)
	ct.done = new(atomic.Bool)
	ct.val = config
	return ct
}
