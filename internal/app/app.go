package app

import (
	"context"
	"sync"

	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/internal/app/shim"
	"git.gensokyo.uk/security/fortify/internal/linux"
)

type App interface {
	// ID returns a copy of App's unique ID.
	ID() fst.ID
	// Run sets up the system and runs the App.
	Run(ctx context.Context, rs *RunState) error

	Seal(config *fst.Config) error
	String() string
}

type RunState struct {
	// Start is true if fsu is successfully started.
	Start bool
	// ExitCode is the value returned by shim.
	ExitCode int
	// WaitErr is error returned by the underlying wait syscall.
	WaitErr error
}

type app struct {
	// application unique identifier
	id *fst.ID
	// operating system interface
	os linux.System
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

func New(os linux.System) (App, error) {
	a := new(app)
	a.id = new(fst.ID)
	a.os = os
	return a, fst.NewAppID(a.id)
}
