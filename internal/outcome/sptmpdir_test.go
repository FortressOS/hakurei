package outcome

import (
	"testing"

	"hakurei.app/container"
	"hakurei.app/container/fhs"
	"hakurei.app/container/std"
	"hakurei.app/container/stub"
	"hakurei.app/hst"
	"hakurei.app/internal/acl"
	"hakurei.app/internal/system"
)

func TestSpTmpdirOp(t *testing.T) {
	t.Parallel()

	checkOpBehaviour(t, []opBehaviourTestCase{
		{"success", func(bool, bool) outcomeOp {
			return spTmpdirOp{}
		}, hst.Template, nil, []stub.Call{
			// this op configures the system state and does not make calls during toSystem
		}, newI().
			Ensure(m("/proc/nonexistent/tmp/hakurei.0/tmpdir"), 0700).
			UpdatePermType(system.User, m("/proc/nonexistent/tmp/hakurei.0/tmpdir"), acl.Execute).
			Ensure(m("/proc/nonexistent/tmp/hakurei.0/tmpdir/9"), 01700).
			UpdatePermType(system.User, m("/proc/nonexistent/tmp/hakurei.0/tmpdir/9"), acl.Read, acl.Write, acl.Execute), nil, nil, insertsOps(nil), []stub.Call{
			// this op configures the container state and does not make calls during toContainer
		}, &container.Params{
			Ops: new(container.Ops).
				Bind(m("/proc/nonexistent/tmp/hakurei.0/tmpdir/9"), fhs.AbsTmp, std.BindWritable),
		}, nil, nil},
	})
}
