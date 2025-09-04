package system

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"
	"sync"
	"syscall"

	"hakurei.app/container"
	"hakurei.app/system/dbus"
)

var (
	ErrDBusConfig = errors.New("dbus config not supplied")
)

// MustProxyDBus calls ProxyDBus and panics if an error is returned.
func (sys *I) MustProxyDBus(sessionPath string, session *dbus.Config, systemPath string, system *dbus.Config) *I {
	if _, err := sys.ProxyDBus(session, system, sessionPath, systemPath); err != nil {
		panic(err.Error())
	} else {
		return sys
	}
}

// ProxyDBus finalises configuration and appends [DBusProxyOp] to [I].
func (sys *I) ProxyDBus(session, system *dbus.Config, sessionPath, systemPath string) (func(), error) {
	d := new(DBusProxyOp)

	// session bus is required as otherwise this is effectively a very expensive noop
	if session == nil {
		return nil, newOpErrorMessage("dbus", ErrDBusConfig,
			"attempted to create message bus proxy args without session bus config", false)
	}

	// system bus is optional
	d.system = system != nil

	var sessionBus, systemBus dbus.ProxyPair
	sessionBus[0], systemBus[0] = dbus.Address()
	sessionBus[1], systemBus[1] = sessionPath, systemPath
	d.out = &linePrefixWriter{println: log.Println, prefix: "(dbus) ", msg: new(strings.Builder)}
	if final, err := dbus.Finalise(sessionBus, systemBus, session, system); err != nil {
		if errors.Is(err, syscall.EINVAL) {
			return nil, newOpErrorMessage("dbus", err,
				"message bus proxy configuration contains NUL byte", false)
		}
		return nil, newOpErrorMessage("dbus", err,
			fmt.Sprintf("cannot finalise message bus proxy: %v", err), false)
	} else {
		if msg.IsVerbose() {
			msg.Verbose("session bus proxy:", session.Args(sessionBus))
			if system != nil {
				msg.Verbose("system bus proxy:", system.Args(systemBus))
			}

			// this calls the argsWt String method
			msg.Verbose("message bus proxy final args:", final.WriterTo)
		}

		d.final = final
	}

	sys.ops = append(sys.ops, d)
	return d.out.Dump, nil
}

// DBusProxyOp starts xdg-dbus-proxy via [dbus] and terminates it on revert.
// This [Op] is always [Process] scoped.
type DBusProxyOp struct {
	proxy *dbus.Proxy // populated during apply

	final *dbus.Final
	out   *linePrefixWriter
	// whether system bus proxy is enabled
	system bool
}

func (d *DBusProxyOp) Type() Enablement { return Process }

func (d *DBusProxyOp) apply(sys *I) error {
	msg.Verbosef("session bus proxy on %q for upstream %q", d.final.Session[1], d.final.Session[0])
	if d.system {
		msg.Verbosef("system bus proxy on %q for upstream %q", d.final.System[1], d.final.System[0])
	}

	d.proxy = dbus.New(sys.ctx, d.final, d.out)
	if err := d.proxy.Start(); err != nil {
		d.out.Dump()
		return newOpErrorMessage("dbus", err,
			fmt.Sprintf("cannot start message bus proxy: %v", err), false)
	}
	msg.Verbose("starting message bus proxy", d.proxy)
	return nil
}

func (d *DBusProxyOp) revert(*I, *Criteria) error {
	// criteria ignored here since dbus is always process-scoped
	msg.Verbose("terminating message bus proxy")
	d.proxy.Close()
	defer msg.Verbose("message bus proxy exit")
	err := d.proxy.Wait()
	if errors.Is(err, context.Canceled) {
		msg.Verbose("message bus proxy canceled upstream")
		err = nil
	}
	return newOpErrorMessage("dbus", err,
		fmt.Sprintf("message bus proxy error: %v", err), true)
}

func (d *DBusProxyOp) Is(o Op) bool {
	target, ok := o.(*DBusProxyOp)
	return ok && d != nil && target != nil &&
		d.system == target.system &&
		d.final != nil && target.final != nil &&
		d.final.Session == target.final.Session &&
		d.final.System == target.final.System &&
		dbus.EqualAddrEntries(d.final.SessionUpstream, target.final.SessionUpstream) &&
		dbus.EqualAddrEntries(d.final.SystemUpstream, target.final.SystemUpstream) &&
		reflect.DeepEqual(d.final.WriterTo, target.final.WriterTo)
}

func (d *DBusProxyOp) Path() string   { return container.Nonexistent }
func (d *DBusProxyOp) String() string { return d.proxy.String() }

// linePrefixWriter calls println with a prefix for every line written.
type linePrefixWriter struct {
	prefix  string
	println func(v ...any)
	msg     *strings.Builder
	msgbuf  []string

	mu sync.RWMutex
}

func (s *linePrefixWriter) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.write(p, 0)
}

func (s *linePrefixWriter) write(p []byte, a int) (int, error) {
	if i := bytes.IndexByte(p, '\n'); i == -1 {
		n, _ := s.msg.Write(p)
		return a + n, nil
	} else {
		n, _ := s.msg.Write(p[:i])

		// allow container init messages through
		v := s.msg.String()
		if strings.HasPrefix(v, "init: ") {
			s.println(s.prefix + v)
		} else {
			s.msgbuf = append(s.msgbuf, v)
		}

		s.msg.Reset()
		return s.write(p[i+1:], a+n+1)
	}
}

func (s *linePrefixWriter) Dump() {
	s.mu.RLock()
	for _, m := range s.msgbuf {
		s.println(s.prefix + m)
	}
	s.mu.RUnlock()
}
