package app

import (
	"path"

	"git.ophivana.moe/cat/fortify/acl"
	"git.ophivana.moe/cat/fortify/dbus"
	"git.ophivana.moe/cat/fortify/internal/state"
)

const (
	dbusSessionBusAddress = "DBUS_SESSION_BUS_ADDRESS"
	dbusSystemBusAddress  = "DBUS_SYSTEM_BUS_ADDRESS"
)

func (seal *appSeal) shareDBus(config [2]*dbus.Config) error {
	if !seal.et.Has(state.EnableDBus) {
		return nil
	}

	// downstream socket paths
	sessionPath, systemPath := path.Join(seal.share, "bus"), path.Join(seal.share, "system_bus_socket")

	// configure dbus proxy
	if err := seal.sys.ProxyDBus(config[0], config[1], sessionPath, systemPath); err != nil {
		return err
	}

	// share proxy sockets
	sessionInner := path.Join(seal.sys.runtime, "bus")
	seal.sys.bwrap.SetEnv[dbusSessionBusAddress] = "unix:path=" + sessionInner
	seal.sys.bwrap.Bind(sessionPath, sessionInner)
	seal.sys.UpdatePerm(sessionPath, acl.Read, acl.Write)
	if config[1] != nil {
		systemInner := "/run/dbus/system_bus_socket"
		seal.sys.bwrap.SetEnv[dbusSystemBusAddress] = "unix:path=" + systemInner
		seal.sys.bwrap.Bind(systemPath, systemInner)
		seal.sys.UpdatePerm(systemPath, acl.Read, acl.Write)
	}

	return nil
}
