package dbus

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// Proxy holds references to a xdg-dbus-proxy process, and should never be copied.
// Once sealed, configuration changes will no longer be possible and attempting to do so will result in a panic.
type Proxy struct {
	cmd *exec.Cmd

	statP [2]*os.File
	argsP [2]*os.File

	address [2]string
	path    string

	wait  *chan error
	read  *chan error
	ready *chan bool

	seal *string
	lock sync.RWMutex
}

func (p *Proxy) String() string {
	if p.cmd != nil {
		return p.cmd.String()
	}

	if p.seal != nil {
		return *p.seal
	}

	return "(unsealed dbus proxy)"
}

// Seal seals the Proxy instance.
func (p *Proxy) Seal(c *Config) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.seal != nil {
		panic("dbus proxy sealed twice")
	}
	args := c.Args(p.address[0], p.address[1])

	seal := strings.Builder{}
	for _, arg := range args {
		// reject argument strings containing null
		for _, b := range arg {
			if b == '\x00' {
				return errors.New("argument contains null")
			}
		}

		// write null terminated argument
		seal.WriteString(arg)
		seal.WriteByte('\x00')
	}
	v := seal.String()
	p.seal = &v
	return nil
}

// New returns a reference to a new unsealed Proxy.
func New(binPath, address, path string) *Proxy {
	return &Proxy{path: binPath, address: [2]string{address, path}}
}
