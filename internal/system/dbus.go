package system

import (
	"bytes"
	"errors"
	"os"
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

func (sys *I) ProxyDBus(session, system *dbus.Config, sessionPath, systemPath string) (func(f func(msgbuf []string)), error) {
	d := new(DBus)

	// used by waiting goroutine to notify process exit
	d.done = make(chan struct{})

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
	return d.out.F, fmsg.WrapErrorSuffix(d.proxy.Seal(session, system),
		"cannot seal message bus proxy:")
}

type DBus struct {
	proxy *dbus.Proxy

	out *scanToFmsg
	// whether system bus proxy is enabled
	system bool
	// notification from goroutine waiting for dbus.Proxy
	done chan struct{}
}

func (d *DBus) Type() Enablement {
	return Process
}

func (d *DBus) apply(_ *I) error {
	fmsg.VPrintf("session bus proxy on %q for upstream %q", d.proxy.Session()[1], d.proxy.Session()[0])
	if d.system {
		fmsg.VPrintf("system bus proxy on %q for upstream %q", d.proxy.System()[1], d.proxy.System()[0])
	}

	// ready channel passed to dbus package
	ready := make(chan error, 1)

	// background dbus proxy start
	if err := d.proxy.Start(ready, d.out, true, true); err != nil {
		return fmsg.WrapErrorSuffix(err,
			"cannot start message bus proxy:")
	}
	fmsg.VPrintln("starting message bus proxy:", d.proxy)
	if fmsg.Verbose() { // save the extra bwrap arg build when verbose logging is off
		fmsg.VPrintln("message bus proxy bwrap args:", d.proxy.BwrapStatic())
	}

	// background wait for proxy instance and notify completion
	go func() {
		if err := d.proxy.Wait(); err != nil {
			fmsg.Println("message bus proxy exited with error:", err)
			go func() { ready <- err }()
		} else {
			fmsg.VPrintln("message bus proxy exit")
		}

		// ensure socket removal so ephemeral directory is empty at revert
		if err := os.Remove(d.proxy.Session()[1]); err != nil && !errors.Is(err, os.ErrNotExist) {
			fmsg.Println("cannot remove dangling session bus socket:", err)
		}
		if d.system {
			if err := os.Remove(d.proxy.System()[1]); err != nil && !errors.Is(err, os.ErrNotExist) {
				fmsg.Println("cannot remove dangling system bus socket:", err)
			}
		}

		// notify proxy completion
		close(d.done)
	}()

	// ready is not nil if the proxy process faulted
	if err := <-ready; err != nil {
		// note that err here is either an I/O error or a predetermined unexpected behaviour error
		return fmsg.WrapErrorSuffix(err,
			"message bus proxy fault after start:")
	}
	fmsg.VPrintln("message bus proxy ready")

	return nil
}

func (d *DBus) revert(_ *I, _ *Criteria) error {
	// criteria ignored here since dbus is always process-scoped
	fmsg.VPrintln("terminating message bus proxy")

	if err := d.proxy.Close(); err != nil {
		if errors.Is(err, os.ErrClosed) {
			return fmsg.WrapError(err,
				"message bus proxy already closed")
		} else {
			return fmsg.WrapErrorSuffix(err,
				"cannot stop message bus proxy:")
		}
	}

	// block until proxy wait returns
	<-d.done
	return nil
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

func (s *scanToFmsg) F(f func(msgbuf []string)) {
	s.mu.RLock()
	f(s.msgbuf)
	s.mu.RUnlock()
}
