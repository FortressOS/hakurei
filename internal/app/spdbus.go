package app

import (
	"hakurei.app/container/fhs"
	"hakurei.app/hst"
	"hakurei.app/system/acl"
	"hakurei.app/system/dbus"
)

// spDBusOp maintains an xdg-dbus-proxy instance for the container.
type spDBusOp struct {
	// Whether to bind the system bus socket.
	// Populated during toSystem.
	ProxySystem bool
}

func (s *spDBusOp) toSystem(state *outcomeStateSys, config *hst.Config) error {
	if config.SessionBus == nil {
		config.SessionBus = dbus.NewConfig(config.ID, true, true)
	}

	// downstream socket paths
	sessionPath, systemPath := state.instance().Append("bus"), state.instance().Append("system_bus_socket")

	if err := state.sys.ProxyDBus(
		config.SessionBus, config.SystemBus,
		sessionPath, systemPath,
	); err != nil {
		return err
	}

	state.sys.UpdatePerm(sessionPath, acl.Read, acl.Write)
	if config.SystemBus != nil {
		s.ProxySystem = true
		state.sys.UpdatePerm(systemPath, acl.Read, acl.Write)
	}
	return nil
}

func (s *spDBusOp) toContainer(state *outcomeStateParams) error {
	sessionInner := state.runtimeDir.Append("bus")
	state.env["DBUS_SESSION_BUS_ADDRESS"] = "unix:path=" + sessionInner.String()
	state.params.Bind(state.instancePath().Append("bus"), sessionInner, 0)
	if s.ProxySystem {
		systemInner := fhs.AbsRun.Append("dbus/system_bus_socket")
		state.env["DBUS_SYSTEM_BUS_ADDRESS"] = "unix:path=" + systemInner.String()
		state.params.Bind(state.instancePath().Append("system_bus_socket"), systemInner, 0)
	}
	return nil
}
