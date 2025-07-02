package system

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"sync"
	"syscall"

	"hakurei.app/system/dbus"
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

	// session bus is required as otherwise this is effectively a very expensive noop
	if session == nil {
		return nil, msg.WrapErr(ErrDBusConfig,
			"attempted to create message bus proxy args without session bus config")
	}

	// system bus is optional
	d.system = system != nil

	d.sessionBus[0], d.systemBus[0] = dbus.Address()
	d.sessionBus[1], d.systemBus[1] = sessionPath, systemPath
	d.out = &scanToFmsg{msg: new(strings.Builder)}
	if final, err := dbus.Finalise(d.sessionBus, d.systemBus, session, system); err != nil {
		if errors.Is(err, syscall.EINVAL) {
			return nil, msg.WrapErr(err, "message bus proxy configuration contains NUL byte")
		}
		return nil, wrapErrSuffix(err, "cannot finalise message bus proxy:")
	} else {
		if msg.IsVerbose() {
			msg.Verbose("session bus proxy:", session.Args(d.sessionBus))
			if system != nil {
				msg.Verbose("system bus proxy:", system.Args(d.systemBus))
			}

			// this calls the argsWt String method
			msg.Verbose("message bus proxy final args:", final.WriterTo)
		}

		d.final = final
	}

	sys.ops = append(sys.ops, d)
	return d.out.Dump, nil
}

type DBus struct {
	proxy *dbus.Proxy // populated during apply

	final *dbus.Final
	out   *scanToFmsg
	// whether system bus proxy is enabled
	system bool

	sessionBus, systemBus dbus.ProxyPair
}

func (d *DBus) Type() Enablement { return Process }

func (d *DBus) apply(sys *I) error {
	msg.Verbosef("session bus proxy on %q for upstream %q", d.sessionBus[1], d.sessionBus[0])
	if d.system {
		msg.Verbosef("system bus proxy on %q for upstream %q", d.systemBus[1], d.systemBus[0])
	}

	d.proxy = dbus.New(sys.ctx, d.final, d.out)
	if err := d.proxy.Start(); err != nil {
		d.out.Dump()
		return wrapErrSuffix(err,
			"cannot start message bus proxy:")
	}
	msg.Verbose("starting message bus proxy", d.proxy)
	return nil
}

func (d *DBus) revert(*I, *Criteria) error {
	// criteria ignored here since dbus is always process-scoped
	msg.Verbose("terminating message bus proxy")
	d.proxy.Close()
	defer msg.Verbose("message bus proxy exit")
	err := d.proxy.Wait()
	if errors.Is(err, context.Canceled) {
		msg.Verbose("message bus proxy canceled upstream")
		err = nil
	}
	return wrapErrSuffix(err, "message bus proxy error:")
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

		// allow container init messages through
		v := s.msg.String()
		if strings.HasPrefix(v, "init: ") {
			log.Println("(dbus) " + v)
		} else {
			s.msgbuf = append(s.msgbuf, v)
		}

		s.msg.Reset()
		return s.write(p[i+1:], a+n+1)
	}
}

func (s *scanToFmsg) Dump() {
	s.mu.RLock()
	for _, msg := range s.msgbuf {
		log.Println("(dbus) " + msg)
	}
	s.mu.RUnlock()
}
