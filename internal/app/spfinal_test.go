package app

import (
	"syscall"
	"testing"

	"hakurei.app/container"
	"hakurei.app/container/fhs"
	"hakurei.app/container/stub"
	"hakurei.app/hst"
	"hakurei.app/system"
	"hakurei.app/system/acl"
)

func TestSpFinalOp(t *testing.T) {
	checkOpBehaviour(t, []opBehaviourTestCase{
		{"nil extra invalid env", func(bool, bool) outcomeOp {
			return spFinalOp{}
		}, func() *hst.Config {
			c := hst.Template()
			// verify nil check behaviour
			c.ExtraPerms = append(c.ExtraPerms, hst.ExtraPermConfig{})
			// verify toContainer behaviour
			c.Container.Env["="] = "\x00"
			return c
		}, nil, []stub.Call{
			// this op configures the system state and does not make calls during toSystem
		}, newI().
			Ensure(m("/var/lib/hakurei/u0"), 0700).
			UpdatePermType(system.User, m("/var/lib/hakurei/u0"),
				acl.Execute).
			UpdatePermType(system.User, m("/var/lib/hakurei/u0/org.chromium.Chromium"),
				acl.Read, acl.Write, acl.Execute), nil, nil, func(state *outcomeStateParams) {
			state.params.Ops = new(container.Ops)
		}, []stub.Call{
			// this op configures the container state and does not make calls during toContainer
		}, nil, nil, &hst.AppError{
			Step: "flatten environment",
			Err:  syscall.EINVAL,
			Msg:  "invalid environment variable =",
		}},

		{"success", func(bool, bool) outcomeOp {
			return spFinalOp{}
		}, hst.Template, nil, []stub.Call{
			// this op configures the system state and does not make calls during toSystem
		}, newI().
			Ensure(m("/var/lib/hakurei/u0"), 0700).
			UpdatePermType(system.User, m("/var/lib/hakurei/u0"),
				acl.Execute).
			UpdatePermType(system.User, m("/var/lib/hakurei/u0/org.chromium.Chromium"),
				acl.Read, acl.Write, acl.Execute), nil, nil, func(state *outcomeStateParams) {
			state.params.Ops = new(container.Ops)
		}, []stub.Call{
			// this op configures the container state and does not make calls during toContainer
		}, &container.Params{
			Env: []string{
				"GOOGLE_API_KEY=AIzaSyBHDrl33hwRp4rMQY0ziRbj8K9LPA6vUCY",
				"GOOGLE_DEFAULT_CLIENT_ID=77185425430.apps.googleusercontent.com",
				"GOOGLE_DEFAULT_CLIENT_SECRET=OTJgUOQcT7lO7GsGZq2G4IlT",
			},
			Ops: new(container.Ops).Remount(fhs.AbsRoot, syscall.MS_RDONLY),
		}, nil, nil},
	})
}
