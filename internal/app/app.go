package app

import (
	"os/exec"
	"sync"
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
	// underlying user switcher process
	cmd *exec.Cmd
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

	if a.cmd != nil {
		return a.cmd.String()
	}

	if a.seal != nil {
		return "(sealed fortified app as uid " + a.seal.sys.user.Uid + ")"
	}

	return "(unsealed fortified app)"
}

func (a *app) WaitErr() error {
	return a.waitErr
}

func New() (App, error) {
	a := new(app)
	a.id = new(ID)
	return a, newAppID(a.id)
}
