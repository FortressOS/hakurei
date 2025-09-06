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
	sessionBus[0], systemBus[0] = sys.dbusAddress()
	sessionBus[1], systemBus[1] = sessionPath, systemPath
	d.out = &linePrefixWriter{println: log.Println, prefix: "(dbus) ", buf: new(strings.Builder)}
	if final, err := sys.dbusFinalise(sessionBus, systemBus, session, system); err != nil {
		if errors.Is(err, syscall.EINVAL) {
			return nil, newOpErrorMessage("dbus", err,
				"message bus proxy configuration contains NUL byte", false)
		}
		return nil, newOpErrorMessage("dbus", err,
			fmt.Sprintf("cannot finalise message bus proxy: %v", err), false)
	} else {
		if sys.isVerbose() {
			sys.verbose("session bus proxy:", session.Args(sessionBus))
			if system != nil {
				sys.verbose("system bus proxy:", system.Args(systemBus))
			}

			// this calls the argsWt String method
			sys.verbose("message bus proxy final args:", final.WriterTo)
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
	sys.verbosef("session bus proxy on %q for upstream %q", d.final.Session[1], d.final.Session[0])
	if d.system {
		sys.verbosef("system bus proxy on %q for upstream %q", d.final.System[1], d.final.System[0])
	}

	d.proxy = dbus.New(sys.ctx, d.final, d.out)
	if err := sys.dbusProxyStart(d.proxy); err != nil {
		d.out.Dump()
		return newOpErrorMessage("dbus", err,
			fmt.Sprintf("cannot start message bus proxy: %v", err), false)
	}
	sys.verbose("starting message bus proxy", d.proxy)
	return nil
}

func (d *DBusProxyOp) revert(sys *I, _ *Criteria) error {
	// criteria ignored here since dbus is always process-scoped
	sys.verbose("terminating message bus proxy")
	sys.dbusProxyClose(d.proxy)

	exitMessage := "message bus proxy exit"
	defer func() { sys.verbose(exitMessage) }()

	err := sys.dbusProxyWait(d.proxy)
	if errors.Is(err, context.Canceled) {
		exitMessage = "message bus proxy canceled upstream"
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

const (
	// lpwSizeThreshold is the threshold of bytes written to linePrefixWriter which,
	// if reached or exceeded, causes linePrefixWriter to drop all future writes.
	lpwSizeThreshold = 1 << 24
)

// linePrefixWriter calls println with a prefix for every line written.
type linePrefixWriter struct {
	prefix  string
	println func(v ...any)

	n   int
	msg []string
	buf *strings.Builder

	mu sync.RWMutex
}

func (s *linePrefixWriter) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.write(p, 0)
}

func (s *linePrefixWriter) write(p []byte, a int) (int, error) {
	if s.n >= lpwSizeThreshold {
		if len(p) == 0 {
			return a, nil
		}
		return a, syscall.ENOMEM
	}

	if i := bytes.IndexByte(p, '\n'); i == -1 {
		n, _ := s.buf.Write(p)
		s.n += n
		return a + n, nil
	} else {
		n, _ := s.buf.Write(p[:i])
		s.n += n + 1

		v := s.buf.String()
		if strings.HasPrefix(v, "init: ") {
			s.n -= len(v) + 1
			// pass through container init messages
			s.println(s.prefix + v)
		} else {
			s.msg = append(s.msg, v)
		}

		s.buf.Reset()
		return s.write(p[i+1:], a+n+1)
	}
}

func (s *linePrefixWriter) Dump() {
	s.mu.RLock()
	for _, m := range s.msg {
		s.println(s.prefix + m)
	}
	if s.buf != nil && s.buf.Len() != 0 {
		s.println("*" + s.prefix + s.buf.String())
	}
	if s.n >= lpwSizeThreshold {
		s.println("+" + s.prefix + "write threshold reached, output may be incomplete")
	}
	s.mu.RUnlock()
}
