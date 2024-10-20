package dbus

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"git.ophivana.moe/security/fortify/helper"
	"git.ophivana.moe/security/fortify/helper/bwrap"
)

// ProxyName is the file name or path to the proxy program.
// Overriding ProxyName will only affect Proxy instance created after the change.
var ProxyName = "xdg-dbus-proxy"

// Proxy holds references to a xdg-dbus-proxy process, and should never be copied.
// Once sealed, configuration changes will no longer be possible and attempting to do so will result in a panic.
type Proxy struct {
	helper helper.Helper
	bwrap  *bwrap.Config

	name    string
	session [2]string
	system  [2]string

	seal io.WriterTo
	lock sync.RWMutex
}

func (p *Proxy) Session() [2]string {
	return p.session
}

func (p *Proxy) System() [2]string {
	return p.system
}

func (p *Proxy) Sealed() bool {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.seal != nil
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
		return p.helper.Unwrap().String()
	}

	if p.seal != nil {
		return p.seal.(fmt.Stringer).String()
	}

	return "(unsealed dbus proxy)"
}

func (p *Proxy) Bwrap() []string {
	return p.bwrap.Args()
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
func New(session, system [2]string) *Proxy {
	return &Proxy{name: ProxyName, session: session, system: system}
}
