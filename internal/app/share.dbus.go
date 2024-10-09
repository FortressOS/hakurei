package app

import (
	"errors"
	"fmt"
	"os"
	"path"

	"git.ophivana.moe/cat/fortify/acl"
	"git.ophivana.moe/cat/fortify/dbus"
	"git.ophivana.moe/cat/fortify/internal/state"
	"git.ophivana.moe/cat/fortify/internal/verbose"
)

const (
	dbusSessionBusAddress = "DBUS_SESSION_BUS_ADDRESS"
	dbusSystemBusAddress  = "DBUS_SYSTEM_BUS_ADDRESS"
)

var (
	ErrDBusConfig = errors.New("dbus config not supplied")
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

	// resolve upstream bus addresses
	sessionBus[0], systemBus[0] = dbus.Address()

	// create proxy instance
	seal.sys.dbus = dbus.New(sessionBus, systemBus)

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
	ready := make(chan error, 1)
	// used by waiting goroutine to notify process return
	tx.dbusWait = make(chan struct{})

	// background dbus proxy start
	if err := tx.dbus.Start(ready, os.Stderr, true); err != nil {
		return (*StartDBusError)(wrapError(err, "cannot start message bus proxy:", err))
	}
	verbose.Println("starting message bus proxy:", tx.dbus)
	verbose.Println("message bus proxy bwrap args:", tx.dbus.Bwrap())

	// background wait for proxy instance and notify completion
	go func() {
		if err := tx.dbus.Wait(); err != nil {
			fmt.Println("fortify: warn: message bus proxy returned error:", err)
			go func() { ready <- err }()
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

	// ready is not nil if the proxy process faulted
	if err := <-ready; err != nil {
		// note that err here is either an I/O related error or a predetermined unexpected behaviour error
		return (*StartDBusError)(wrapError(err, "message bus proxy fault after start:", err))
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
