package app

import (
	"testing"

	"hakurei.app/container"
	"hakurei.app/container/bits"
	"hakurei.app/container/fhs"
	"hakurei.app/container/stub"
	"hakurei.app/hst"
	"hakurei.app/system"
	"hakurei.app/system/acl"
)

func TestSpRuntimeOp(t *testing.T) {
	t.Parallel()
	config := hst.Template()

	checkOpBehaviour(t, []opBehaviourTestCase{
		{"success", func(bool, bool) outcomeOp {
			return spRuntimeOp{}
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
				Bind(m("/proc/nonexistent/tmp/hakurei.0/runtime/9"), m("/run/user/1000"), bits.BindWritable),
		}, paramsWantEnv(config, map[string]string{
			"XDG_RUNTIME_DIR":   "/run/user/1000",
			"XDG_SESSION_CLASS": "user",
			"XDG_SESSION_TYPE":  "tty",
		}, nil), nil},
	})
}
