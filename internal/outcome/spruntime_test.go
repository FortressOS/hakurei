package outcome

import (
	"testing"

	"hakurei.app/container"
	"hakurei.app/container/fhs"
	"hakurei.app/container/std"
	"hakurei.app/container/stub"
	"hakurei.app/hst"
	"hakurei.app/system"
	"hakurei.app/system/acl"
)

func TestSpRuntimeOp(t *testing.T) {
	t.Parallel()
	config := hst.Template()

	checkOpBehaviour(t, []opBehaviourTestCase{
		{"success zero", func(isShim bool, clearUnexported bool) outcomeOp {
			if !isShim {
				return new(spRuntimeOp)
			}
			op := &spRuntimeOp{sessionTypeTTY}
			if clearUnexported {
				op.SessionType = sessionTypeUnspec
			}
			return op
		}, func() *hst.Config {
			c := hst.Template()
			*c.Enablements = 0
			return c
		}, nil, []stub.Call{
			// this op configures the system state and does not make calls during toSystem
		}, newI().
			Ensure(m("/proc/nonexistent/tmp/hakurei.0/runtime"), 0700).
			UpdatePermType(system.User, m("/proc/nonexistent/tmp/hakurei.0/runtime"), acl.Execute).
			Ensure(m("/proc/nonexistent/tmp/hakurei.0/runtime/9"), 0700).
			UpdatePermType(system.User, m("/proc/nonexistent/tmp/hakurei.0/runtime/9"), acl.Read, acl.Write, acl.Execute), nil, nil, insertsOps(nil), []stub.Call{
			// this op configures the container state and does not make calls during toContainer
		}, &container.Params{
			Ops: new(container.Ops).
				Tmpfs(fhs.AbsRunUser, 1<<12, 0755).
				Bind(m("/proc/nonexistent/tmp/hakurei.0/runtime/9"), m("/run/user/1000"), std.BindWritable),
		}, paramsWantEnv(config, map[string]string{
			"XDG_RUNTIME_DIR":   "/run/user/1000",
			"XDG_SESSION_CLASS": "user",
			"XDG_SESSION_TYPE":  "unspecified",
		}, nil), nil},

		{"success tty", func(isShim, _ bool) outcomeOp {
			if !isShim {
				return new(spRuntimeOp)
			}
			return &spRuntimeOp{sessionTypeTTY}
		}, func() *hst.Config {
			c := hst.Template()
			*c.Enablements = 0
			return c
		}, nil, []stub.Call{
			// this op configures the system state and does not make calls during toSystem
		}, newI().
			Ensure(m("/proc/nonexistent/tmp/hakurei.0/runtime"), 0700).
			UpdatePermType(system.User, m("/proc/nonexistent/tmp/hakurei.0/runtime"), acl.Execute).
			Ensure(m("/proc/nonexistent/tmp/hakurei.0/runtime/9"), 0700).
			UpdatePermType(system.User, m("/proc/nonexistent/tmp/hakurei.0/runtime/9"), acl.Read, acl.Write, acl.Execute), nil, nil, insertsOps(nil), []stub.Call{
			// this op configures the container state and does not make calls during toContainer
		}, &container.Params{
			Ops: new(container.Ops).
				Tmpfs(fhs.AbsRunUser, 1<<12, 0755).
				Bind(m("/proc/nonexistent/tmp/hakurei.0/runtime/9"), m("/run/user/1000"), std.BindWritable),
		}, paramsWantEnv(config, map[string]string{
			"XDG_RUNTIME_DIR":   "/run/user/1000",
			"XDG_SESSION_CLASS": "user",
			"XDG_SESSION_TYPE":  "tty",
		}, nil), nil},

		{"success x11", func(isShim, _ bool) outcomeOp {
			if !isShim {
				return new(spRuntimeOp)
			}
			return &spRuntimeOp{sessionTypeX11}
		}, func() *hst.Config {
			c := hst.Template()
			*c.Enablements = hst.Enablements(hst.EX11)
			return c
		}, nil, []stub.Call{
			// this op configures the system state and does not make calls during toSystem
		}, newI().
			Ensure(m("/proc/nonexistent/tmp/hakurei.0/runtime"), 0700).
			UpdatePermType(system.User, m("/proc/nonexistent/tmp/hakurei.0/runtime"), acl.Execute).
			Ensure(m("/proc/nonexistent/tmp/hakurei.0/runtime/9"), 0700).
			UpdatePermType(system.User, m("/proc/nonexistent/tmp/hakurei.0/runtime/9"), acl.Read, acl.Write, acl.Execute), nil, nil, insertsOps(nil), []stub.Call{
			// this op configures the container state and does not make calls during toContainer
		}, &container.Params{
			Ops: new(container.Ops).
				Tmpfs(fhs.AbsRunUser, 1<<12, 0755).
				Bind(m("/proc/nonexistent/tmp/hakurei.0/runtime/9"), m("/run/user/1000"), std.BindWritable),
		}, paramsWantEnv(config, map[string]string{
			"XDG_RUNTIME_DIR":   "/run/user/1000",
			"XDG_SESSION_CLASS": "user",
			"XDG_SESSION_TYPE":  "x11",
		}, nil), nil},

		{"success", func(isShim, _ bool) outcomeOp {
			if !isShim {
				return new(spRuntimeOp)
			}
			return &spRuntimeOp{sessionTypeWayland}
		}, hst.Template, nil, []stub.Call{
			// this op configures the system state and does not make calls during toSystem
		}, newI().
			Ensure(m("/proc/nonexistent/tmp/hakurei.0/runtime"), 0700).
			UpdatePermType(system.User, m("/proc/nonexistent/tmp/hakurei.0/runtime"), acl.Execute).
			Ensure(m("/proc/nonexistent/tmp/hakurei.0/runtime/9"), 0700).
			UpdatePermType(system.User, m("/proc/nonexistent/tmp/hakurei.0/runtime/9"), acl.Read, acl.Write, acl.Execute), nil, nil, insertsOps(nil), []stub.Call{
			// this op configures the container state and does not make calls during toContainer
		}, &container.Params{
			Ops: new(container.Ops).
				Tmpfs(fhs.AbsRunUser, 1<<12, 0755).
				Bind(m("/proc/nonexistent/tmp/hakurei.0/runtime/9"), m("/run/user/1000"), std.BindWritable),
		}, paramsWantEnv(config, map[string]string{
			"XDG_RUNTIME_DIR":   "/run/user/1000",
			"XDG_SESSION_CLASS": "user",
			"XDG_SESSION_TYPE":  "wayland",
		}, nil), nil},
	})
}
