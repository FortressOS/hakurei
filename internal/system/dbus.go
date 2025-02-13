package system

import (
	"bytes"
	"errors"
	"strings"
	"sync"

	"git.gensokyo.uk/security/fortify/dbus"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
)

var (
	ErrDBusConfig = errors.New("dbus config not supplied")
)

func (sys *I) MustProxyDBus(sessionPath string, session *dbus.Config, systemPath string, system *dbus.Config) *I {
	if _, err := sys.ProxyDBus(session, system, sessionPath, systemPath); err != nil {
		panic(err.Error())
	} else {
		return sys
	}
}

func (sys *I) ProxyDBus(session, system *dbus.Config, sessionPath, systemPath string) (func(), error) {
	d := new(DBus)

	// session bus is mandatory
	if session == nil {
		return nil, fmsg.WrapError(ErrDBusConfig,
			"attempted to seal message bus proxy without session bus config")
	}

	// system bus is optional
	d.system = system != nil

	// upstream address, downstream socket path
	var sessionBus, systemBus [2]string

	// resolve upstream bus addresses
	sessionBus[0], systemBus[0] = dbus.Address()

	// set paths from caller
	sessionBus[1], systemBus[1] = sessionPath, systemPath

	// create proxy instance
	d.proxy = dbus.New(sessionBus, systemBus)

	defer func() {
		if fmsg.Verbose() && d.proxy.Sealed() {
			fmsg.VPrintln("sealed session proxy", session.Args(sessionBus))
			if system != nil {
				fmsg.VPrintln("sealed system proxy", system.Args(systemBus))
			}
			fmsg.VPrintln("message bus proxy final args:", d.proxy)
		}
	}()

	// queue operation
	sys.ops = append(sys.ops, d)

	// seal dbus proxy
	d.out = &scanToFmsg{msg: new(strings.Builder)}
	return d.out.Dump, fmsg.WrapErrorSuffix(d.proxy.Seal(session, system),
		"cannot seal message bus proxy:")
}

type DBus struct {
	proxy *dbus.Proxy

	out *scanToFmsg
	// whether system bus proxy is enabled
	system bool
}

func (d *DBus) Type() Enablement {
	return Process
}

func (d *DBus) apply(sys *I) error {
	fmsg.VPrintf("session bus proxy on %q for upstream %q", d.proxy.Session()[1], d.proxy.Session()[0])
	if d.system {
		fmsg.VPrintf("system bus proxy on %q for upstream %q", d.proxy.System()[1], d.proxy.System()[0])
	}

	// this starts the process and blocks until ready
	if err := d.proxy.Start(sys.ctx, d.out, true); err != nil {
		d.out.Dump()
		return fmsg.WrapErrorSuffix(err,
			"cannot start message bus proxy:")
	}
	fmsg.VPrintln("starting message bus proxy:", d.proxy)
	return nil
}

func (d *DBus) revert(_ *I, _ *Criteria) error {
	// criteria ignored here since dbus is always process-scoped
	fmsg.VPrintln("terminating message bus proxy")
	d.proxy.Close()
	defer fmsg.VPrintln("message bus proxy exit")
	return fmsg.WrapErrorSuffix(d.proxy.Wait(), "message bus proxy error:")
}

func (d *DBus) Is(o Op) bool {
	d0, ok := o.(*DBus)
	return ok && d0 != nil &&
		((d.proxy == nil && d0.proxy == nil) ||
			(d.proxy != nil && d0.proxy != nil && d.proxy.String() == d0.proxy.String()))
}

func (d *DBus) Path() string {
	return "(dbus proxy)"
}

func (d *DBus) String() string {
	return d.proxy.String()
}

type scanToFmsg struct {
	msg    *strings.Builder
	msgbuf []string

	mu sync.RWMutex
}

func (s *scanToFmsg) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.write(p, 0)
}

func (s *scanToFmsg) write(p []byte, a int) (int, error) {
	if i := bytes.IndexByte(p, '\n'); i == -1 {
		n, _ := s.msg.Write(p)
		return a + n, nil
	} else {
		n, _ := s.msg.Write(p[:i])
		s.msgbuf = append(s.msgbuf, s.msg.String())
		s.msg.Reset()
		return s.write(p[i+1:], a+n+1)
	}
}

func (s *scanToFmsg) Dump() {
	s.mu.RLock()
	for _, msg := range s.msgbuf {
		fmsg.Println(msg)
	}
	s.mu.RUnlock()
}
