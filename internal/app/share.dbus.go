package app

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"

	"git.ophivana.moe/cat/fortify/acl"
	"git.ophivana.moe/cat/fortify/dbus"
	"git.ophivana.moe/cat/fortify/internal/state"
	"git.ophivana.moe/cat/fortify/internal/verbose"
)

const (
	dbusSessionBusAddress = "DBUS_SESSION_BUS_ADDRESS"
	dbusSystemBusAddress  = "DBUS_SYSTEM_BUS_ADDRESS"

	xdgDBusProxy = "xdg-dbus-proxy"
)

var (
	ErrDBusConfig = errors.New("dbus config not supplied")
	ErrDBusProxy  = errors.New(xdgDBusProxy + " not found")
	ErrDBusFault  = errors.New(xdgDBusProxy + " did not start correctly")
)

type (
	SealDBusError   BaseError
	LookupDBusError BaseError
	StartDBusError  BaseError
	CloseDBusError  BaseError
)

func (seal *appSeal) shareDBus(config [2]*dbus.Config) error {
	if !seal.et.Has(state.EnableDBus) {
		return nil
	}

	// session bus is mandatory
	if config[0] == nil {
		return (*SealDBusError)(wrapError(ErrDBusConfig, "attempted to seal session bus proxy with nil config"))
	}

	// system bus is optional
	seal.sys.dbusSystem = config[1] != nil

	// upstream address, downstream socket path
	var sessionBus, systemBus [2]string

	// downstream socket paths
	sessionBus[1] = path.Join(seal.share, "bus")
	systemBus[1] = path.Join(seal.share, "system_bus_socket")

	// resolve upstream session bus address
	if addr, ok := os.LookupEnv(dbusSessionBusAddress); !ok {
		// fall back to default format
		sessionBus[0] = fmt.Sprintf("unix:path=/run/user/%d/bus", os.Getuid())
	} else {
		sessionBus[0] = addr
	}

	// resolve upstream system bus address
	if addr, ok := os.LookupEnv(dbusSystemBusAddress); !ok {
		// fall back to default hardcoded value
		systemBus[0] = "unix:path=/run/dbus/system_bus_socket"
	} else {
		systemBus[0] = addr
	}

	// look up proxy program path for dbus.New
	if b, err := exec.LookPath(xdgDBusProxy); err != nil {
		return (*LookupDBusError)(wrapError(ErrDBusProxy, xdgDBusProxy, "not found"))
	} else {
		// create proxy instance
		seal.sys.dbus = dbus.New(b, sessionBus, systemBus)
	}

	// seal dbus proxy
	if err := seal.sys.dbus.Seal(config[0], config[1]); err != nil {
		return (*SealDBusError)(wrapError(err, "cannot seal message bus proxy:", err))
	}

	// store addresses for cleanup and logging
	seal.sys.dbusAddr = &[2][2]string{sessionBus, systemBus}

	// share proxy sockets
	seal.appendEnv(dbusSessionBusAddress, "unix:path="+sessionBus[1])
	seal.sys.updatePerm(sessionBus[1], acl.Read, acl.Write)
	if seal.sys.dbusSystem {
		seal.appendEnv(dbusSystemBusAddress, "unix:path="+systemBus[1])
		seal.sys.updatePerm(systemBus[1], acl.Read, acl.Write)
	}

	return nil
}

func (tx *appSealTx) startDBus() error {
	// ready channel passed to dbus package
	ready := make(chan bool, 1)
	// used by waiting goroutine to notify process return
	tx.dbusWait = make(chan struct{})

	// background dbus proxy start
	if err := tx.dbus.Start(&ready); err != nil {
		return (*StartDBusError)(wrapError(err, "cannot start message bus proxy:", err))
	}
	verbose.Println("starting message bus proxy:", tx.dbus)

	// background wait for proxy instance and notify completion
	go func() {
		if err := tx.dbus.Wait(); err != nil {
			fmt.Println("fortify: warn: message bus proxy returned error:", err)
		} else {
			verbose.Println("message bus proxy exit")
		}

		// ensure socket removal so ephemeral directory is empty at revert
		if err := os.Remove(tx.dbusAddr[0][1]); err != nil && !errors.Is(err, os.ErrNotExist) {
			fmt.Println("fortify: cannot remove dangling session bus socket:", err)
		}
		if tx.dbusSystem {
			if err := os.Remove(tx.dbusAddr[1][1]); err != nil && !errors.Is(err, os.ErrNotExist) {
				fmt.Println("fortify: cannot remove dangling system bus socket:", err)
			}
		}

		// notify proxy completion
		tx.dbusWait <- struct{}{}
	}()

	// ready is false if the proxy process faulted
	if !<-ready {
		return (*StartDBusError)(wrapError(ErrDBusFault, "message bus proxy failed"))
	}
	verbose.Println("message bus proxy ready")

	return nil
}

func (tx *appSealTx) stopDBus() error {
	if err := tx.dbus.Close(); err != nil {
		if errors.Is(err, os.ErrClosed) {
			return (*CloseDBusError)(wrapError(err, "message bus proxy already closed"))
		} else {
			return (*CloseDBusError)(wrapError(err, "cannot close message bus proxy:", err))
		}
	}

	// block until proxy wait returns
	<-tx.dbusWait
	return nil
}
