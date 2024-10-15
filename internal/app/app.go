package app

import (
	"net"
	"os/exec"
	"sync"
)

type App interface {
	Seal(config *Config) error
	Start() error
	Wait() (int, error)
	WaitErr() error
	String() string
}

type app struct {
	// child process related information
	seal *appSeal
	// underlying fortified child process
	cmd *exec.Cmd
	// wayland connection if wayland mediation is enabled
	wayland *net.UnixConn
	// error returned waiting for process
	wait error

	lock sync.RWMutex
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
	return a.wait
}

func New() App {
	return new(app)
}
