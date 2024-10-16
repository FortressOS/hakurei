package system

import (
	"errors"
	"fmt"
	"os"

	"git.ophivana.moe/cat/fortify/dbus"
	"git.ophivana.moe/cat/fortify/internal/fmsg"
	"git.ophivana.moe/cat/fortify/internal/verbose"
)

var (
	ErrDBusConfig = errors.New("dbus config not supplied")
)

func (sys *I) ProxyDBus(session, system *dbus.Config, sessionPath, systemPath string) error {
	d := new(DBus)

	// used by waiting goroutine to notify process exit
	d.done = make(chan struct{})

	// session bus is mandatory
	if session == nil {
		return fmsg.WrapError(ErrDBusConfig,
			"attempted to seal message bus proxy without session bus config")
	}

	// system bus is optional
	d.system = system == nil

	// upstream address, downstream socket path
	var sessionBus, systemBus [2]string

	// resolve upstream bus addresses
	sessionBus[0], systemBus[0] = dbus.Address()

	// set paths from caller
	sessionBus[1], systemBus[1] = sessionPath, systemPath

	// create proxy instance
	d.proxy = dbus.New(sessionBus, systemBus)

	defer func() {
		if verbose.Get() && d.proxy.Sealed() {
			verbose.Println("sealed session proxy", session.Args(sessionBus))
			if system != nil {
				verbose.Println("sealed system proxy", system.Args(systemBus))
			}
			verbose.Println("message bus proxy final args:", d.proxy)
		}
	}()

	// queue operation
	sys.ops = append(sys.ops, d)

	// seal dbus proxy
	return fmsg.WrapErrorSuffix(d.proxy.Seal(session, system),
		"cannot seal message bus proxy:")
}

type DBus struct {
	proxy *dbus.Proxy

	// whether system bus proxy is enabled
	system bool
	// notification from goroutine waiting for dbus.Proxy
	done chan struct{}
}

func (d *DBus) Type() Enablement {
	return Process
}

func (d *DBus) apply(_ *I) error {
	verbose.Printf("session bus proxy on %q for upstream %q\n", d.proxy.Session()[1], d.proxy.Session()[0])
	if d.system {
		verbose.Printf("system bus proxy on %q for upstream %q\n", d.proxy.System()[1], d.proxy.System()[0])
	}

	// ready channel passed to dbus package
	ready := make(chan error, 1)

	// background dbus proxy start
	if err := d.proxy.Start(ready, os.Stderr, true); err != nil {
		return fmsg.WrapErrorSuffix(err,
			"cannot start message bus proxy:")
	}
	verbose.Println("starting message bus proxy:", d.proxy)
	if verbose.Get() { // save the extra bwrap arg build when verbose logging is off
		verbose.Println("message bus proxy bwrap args:", d.proxy.Bwrap())
	}

	// background wait for proxy instance and notify completion
	go func() {
		if err := d.proxy.Wait(); err != nil {
			fmt.Println("fortify: message bus proxy exited with error:", err)
			go func() { ready <- err }()
		} else {
			verbose.Println("message bus proxy exit")
		}

		// ensure socket removal so ephemeral directory is empty at revert
		if err := os.Remove(d.proxy.Session()[1]); err != nil && !errors.Is(err, os.ErrNotExist) {
			fmt.Println("fortify: cannot remove dangling session bus socket:", err)
		}
		if d.system {
			if err := os.Remove(d.proxy.System()[1]); err != nil && !errors.Is(err, os.ErrNotExist) {
				fmt.Println("fortify: cannot remove dangling system bus socket:", err)
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
	verbose.Println("message bus proxy ready")

	return nil
}

func (d *DBus) revert(_ *I, _ *Criteria) error {
	// criteria ignored here since dbus is always process-scoped
	verbose.Println("terminating message bus proxy")

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
	return ok && d0 != nil && *d == *d0
}

func (d *DBus) Path() string {
	return "(dbus proxy)"
}

func (d *DBus) String() string {
	return d.proxy.String()
}
