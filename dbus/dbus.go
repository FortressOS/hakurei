package dbus

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"git.ophivana.moe/cat/fortify/helper"
)

// Proxy holds references to a xdg-dbus-proxy process, and should never be copied.
// Once sealed, configuration changes will no longer be possible and attempting to do so will result in a panic.
type Proxy struct {
	helper *helper.Helper

	path    string
	session [2]string
	system  [2]string

	seal io.WriterTo
	lock sync.RWMutex
}

var (
	ErrConfig = errors.New("no configuration to seal")
)

func (p *Proxy) String() string {
	if p == nil {
		return "(invalid dbus proxy)"
	}

	p.lock.RLock()
	defer p.lock.RUnlock()

	if p.helper != nil {
		return p.helper.String()
	}

	if p.seal != nil {
		return p.seal.(fmt.Stringer).String()
	}

	return "(unsealed dbus proxy)"
}

// Seal seals the Proxy instance.
func (p *Proxy) Seal(session, system *Config) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.seal != nil {
		panic("dbus proxy sealed twice")
	}

	if session == nil && system == nil {
		return ErrConfig
	}

	var args []string
	if session != nil {
		args = append(args, session.Args(p.session)...)
	}
	if system != nil {
		args = append(args, system.Args(p.system)...)
	}
	if seal, err := helper.NewCheckedArgs(args); err != nil {
		return err
	} else {
		p.seal = seal
	}

	return nil
}

// New returns a reference to a new unsealed Proxy.
func New(binPath string, session, system [2]string) *Proxy {
	return &Proxy{path: binPath, session: session, system: system}
}
