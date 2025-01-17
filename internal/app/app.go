package app

import (
	"context"
	"sync"
	"sync/atomic"

	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/internal/linux"
	"git.gensokyo.uk/security/fortify/internal/proc/priv/shim"
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
