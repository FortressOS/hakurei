package app

import (
	"sync"

	"git.ophivana.moe/security/fortify/cmd/fshim/ipc/shim"
	"git.ophivana.moe/security/fortify/internal/linux"
)

type App interface {
	// ID returns a copy of App's unique ID.
	ID() ID
	// Start sets up the system and starts the App.
	Start() error
	// Wait waits for App's process to exit and reverts system setup.
	Wait() (int, error)
	// WaitErr returns error returned by the underlying wait syscall.
	WaitErr() error

	Seal(config *Config) error
	String() string
}

type app struct {
	// application unique identifier
	id *ID
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

func (a *app) ID() ID {
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
		return "(sealed fortified app as uid " + a.seal.sys.user.Uid + ")"
	}

	return "(unsealed fortified app)"
}

func (a *app) WaitErr() error {
	return a.waitErr
}

func New(os linux.System) (App, error) {
	a := new(app)
	a.id = new(ID)
	a.os = os
	return a, newAppID(a.id)
}
