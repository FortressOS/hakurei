package app

import (
	"encoding/gob"

	"hakurei.app/container/fhs"
	"hakurei.app/hst"
	"hakurei.app/system/acl"
	"hakurei.app/system/dbus"
)

func init() { gob.Register(new(spDBusOp)) }

// spDBusOp maintains an xdg-dbus-proxy instance for the container.
type spDBusOp struct {
	// Whether to bind the system bus socket.
	// Populated during toSystem.
	ProxySystem bool
}

func (s *spDBusOp) toSystem(state *outcomeStateSys) error {
	if state.et&hst.EDBus == 0 {
		return errNotEnabled
	}

	if state.sessionBus == nil {
		state.sessionBus = dbus.NewConfig(state.appId, true, true)
	}

	// downstream socket paths
	sessionPath, systemPath := state.instance().Append("bus"), state.instance().Append("system_bus_socket")

	if err := state.sys.ProxyDBus(
		state.sessionBus, state.systemBus,
		sessionPath, systemPath,
	); err != nil {
		return err
	}

	state.sys.UpdatePerm(sessionPath, acl.Read, acl.Write)
	if state.systemBus != nil {
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
