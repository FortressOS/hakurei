package outcome

import (
	"testing"

	"hakurei.app/container"
	"hakurei.app/container/stub"
	"hakurei.app/hst"
	"hakurei.app/internal/acl"
	"hakurei.app/internal/system"
	"hakurei.app/internal/wayland"
)

func TestSpWaylandOp(t *testing.T) {
	t.Parallel()
	config := hst.Template()

	checkOpBehaviour(t, []opBehaviourTestCase{
		{"not enabled", func(bool, bool) outcomeOp {
			return new(spWaylandOp)
		}, func() *hst.Config {
			c := hst.Template()
			*c.Enablements = 0
			return c
		}, nil, nil, nil, nil, errNotEnabled, nil, nil, nil, nil, nil},

		{"success notAbs defaultAppId", func(bool, bool) outcomeOp {
			return new(spWaylandOp)
		}, func() *hst.Config {
			c := hst.Template()
			c.ID = ""
			return c
		}, nil, []stub.Call{
			call("lookupEnv", stub.ExpectArgs{"WAYLAND_DISPLAY"}, "wayland-1", nil),
		}, newI().
			// state.instance
			Ephemeral(system.Process, m(wantInstancePrefix), 0711).
			// toSystem
			Wayland(
				m(wantInstancePrefix+"/wayland"),
				m(wantRuntimePath+"/wayland-1"),
				"app.hakurei."+wantAutoEtcPrefix,
				wantAutoEtcPrefix,
			), sysUsesInstance(nil), nil, insertsOps(afterSpRuntimeOp(nil)), []stub.Call{
			// this op configures the container state and does not make calls during toContainer
		}, &container.Params{
			Ops: new(container.Ops).
				Bind(m(wantInstancePrefix+"/wayland"), m("/run/user/1000/wayland-0"), 0),
		}, paramsWantEnv(config, map[string]string{
			wayland.Display: wayland.FallbackName,
		}, nil), nil},

		{"success direct", func(isShim, _ bool) outcomeOp {
			if !isShim {
				return new(spWaylandOp)
			}
			return &spWaylandOp{SocketPath: m("/proc/nonexistent/wayland")}
		}, func() *hst.Config {
			c := hst.Template()
			c.DirectWayland = true
			return c
		}, nil, []stub.Call{
			call("lookupEnv", stub.ExpectArgs{"WAYLAND_DISPLAY"}, "/proc/nonexistent/wayland", nil),
			call("verbose", stub.ExpectArgs{[]any{"direct wayland access, PROCEED WITH CAUTION"}}, nil, nil),
		}, newI().
			// state.ensureRuntimeDir
			Ensure(m(wantRuntimePath), 0700).
			UpdatePermType(system.User, m(wantRuntimePath), acl.Execute).
			Ensure(m(wantRunDirPath), 0700).
			UpdatePermType(system.User, m(wantRunDirPath), acl.Execute).
			// toSystem
			UpdatePermType(hst.EWayland, m("/proc/nonexistent/wayland"), acl.Read, acl.Write, acl.Execute), nil, nil, insertsOps(afterSpRuntimeOp(nil)), []stub.Call{
			// this op configures the container state and does not make calls during toContainer
		}, &container.Params{
			Ops: new(container.Ops).
				Bind(m("/proc/nonexistent/wayland"), m("/run/user/1000/wayland-0"), 0),
		}, paramsWantEnv(config, map[string]string{
			wayland.Display: wayland.FallbackName,
		}, nil), nil},

		{"success", func(bool, bool) outcomeOp {
			return new(spWaylandOp)
		}, hst.Template, nil, []stub.Call{
			call("lookupEnv", stub.ExpectArgs{"WAYLAND_DISPLAY"}, nil, nil),
			call("verbose", stub.ExpectArgs{[]any{"WAYLAND_DISPLAY is not set, assuming wayland-0"}}, nil, nil),
		}, newI().
			// state.instance
			Ephemeral(system.Process, m(wantInstancePrefix), 0711).
			// toSystem
			Wayland(
				m(wantInstancePrefix+"/wayland"),
				m(wantRuntimePath+"/"+wayland.FallbackName),
				"org.chromium.Chromium",
				wantAutoEtcPrefix,
			), sysUsesInstance(nil), nil, insertsOps(afterSpRuntimeOp(nil)), []stub.Call{
			// this op configures the container state and does not make calls during toContainer
		}, &container.Params{
			Ops: new(container.Ops).
				Bind(m(wantInstancePrefix+"/wayland"), m("/run/user/1000/wayland-0"), 0),
		}, paramsWantEnv(config, map[string]string{
			wayland.Display: wayland.FallbackName,
		}, nil), nil},
	})
}
