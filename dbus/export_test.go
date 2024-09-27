package dbus

import "io"

// CompareTestNew provides TestNew with comparison access to unexported Proxy fields.
func (p *Proxy) CompareTestNew(path string, session, system [2]string) bool {
	return path == p.path && session == p.session && system == p.system
}

// AccessTestProxySeal provides TestProxy_Seal with access to unexported Proxy seal field.
func (p *Proxy) AccessTestProxySeal() io.WriterTo {
	return p.seal
}
