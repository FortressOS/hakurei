package app

import (
	"encoding/gob"

	"hakurei.app/container/check"
	"hakurei.app/hst"
	"hakurei.app/system/acl"
	"hakurei.app/system/wayland"
)

func init() { gob.Register(new(spWaylandOp)) }

// spWaylandOp exports the Wayland display server to the container.
type spWaylandOp struct {
	// Path to host wayland socket. Populated during toSystem if DirectWayland is true.
	SocketPath *check.Absolute
}

func (s *spWaylandOp) toSystem(state *outcomeStateSys) error {
	// outer wayland socket (usually `/run/user/%d/wayland-%d`)
	var socketPath *check.Absolute
	if name, ok := state.k.lookupEnv(wayland.WaylandDisplay); !ok {
		state.msg.Verbose(wayland.WaylandDisplay + " is not set, assuming " + wayland.FallbackName)
		socketPath = state.sc.RuntimePath.Append(wayland.FallbackName)
	} else if a, err := check.NewAbs(name); err != nil {
		socketPath = state.sc.RuntimePath.Append(name)
	} else {
		socketPath = a
	}

	if !state.config.DirectWayland { // set up security-context-v1
		appID := state.config.ID
		if appID == "" {
			// use instance ID in case app id is not set
			appID = "app.hakurei." + state.id.String()
		}
		// downstream socket paths
		state.sys.Wayland(state.instance().Append("wayland"), socketPath, appID, state.id.String())
	} else { // bind mount wayland socket (insecure)
		state.msg.Verbose("direct wayland access, PROCEED WITH CAUTION")
		state.ensureRuntimeDir()
		s.SocketPath = socketPath
		state.sys.UpdatePermType(hst.EWayland, socketPath, acl.Read, acl.Write, acl.Execute)
	}
	return nil
}

func (s *spWaylandOp) toContainer(state *outcomeStateParams) error {
	innerPath := state.runtimeDir.Append(wayland.FallbackName)
	state.env[wayland.WaylandDisplay] = wayland.FallbackName
	if s.SocketPath == nil {
		state.params.Bind(state.instancePath().Append("wayland"), innerPath, 0)
	} else {
		state.params.Bind(s.SocketPath, innerPath, 0)
	}
	return nil
}
