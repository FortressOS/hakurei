package outcome

import (
	"encoding/gob"

	"hakurei.app/container/check"
	"hakurei.app/hst"
	"hakurei.app/internal/acl"
	"hakurei.app/internal/wayland"
)

func init() { gob.Register(new(spWaylandOp)) }

// spWaylandOp exports the Wayland display server to the container.
// Runs after spRuntimeOp.
type spWaylandOp struct {
	// Path to host wayland socket. Populated during toSystem if DirectWayland is true.
	SocketPath *check.Absolute
}

func (s *spWaylandOp) toSystem(state *outcomeStateSys) error {
	if state.et&hst.EWayland == 0 {
		return errNotEnabled
	}

	// outer wayland socket (usually `/run/user/%d/wayland-%d`)
	var socketPath *check.Absolute
	if name, ok := state.k.lookupEnv(wayland.Display); !ok {
		state.msg.Verbose(wayland.Display + " is not set, assuming " + wayland.FallbackName)
		socketPath = state.sc.RuntimePath.Append(wayland.FallbackName)
	} else if a, err := check.NewAbs(name); err != nil {
		socketPath = state.sc.RuntimePath.Append(name)
	} else {
		socketPath = a
	}

	if !state.directWayland { // set up security-context-v1
		appId := state.appId
		if appId == "" {
			// use instance ID in case app id is not set
			appId = "app.hakurei." + state.id.String()
		}
		// downstream socket paths
		state.sys.Wayland(state.instance().Append("wayland"), socketPath, appId, state.id.String())
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
	state.env[wayland.Display] = wayland.FallbackName
	if s.SocketPath == nil {
		state.params.Bind(state.instancePath().Append("wayland"), innerPath, 0)
	} else {
		state.params.Bind(s.SocketPath, innerPath, 0)
	}
	return nil
}
