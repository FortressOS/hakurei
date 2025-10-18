package app

import (
	"os"
	"testing"

	"hakurei.app/container"
	"hakurei.app/container/stub"
	"hakurei.app/hst"
	"hakurei.app/system/acl"
)

func TestSpX11Op(t *testing.T) {
	t.Parallel()
	config := hst.Template()

	checkOpBehaviour(t, []opBehaviourTestCase{
		{"not enabled", func(bool, bool) outcomeOp {
			return new(spX11Op)
		}, hst.Template, nil, nil, nil, nil, errNotEnabled, nil, nil, nil, nil, nil},

		{"lookupEnv", func(bool, bool) outcomeOp {
			return new(spX11Op)
		}, func() *hst.Config {
			c := hst.Template()
			*c.Enablements |= hst.Enablements(hst.EX11)
			return c
		}, nil, []stub.Call{
			call("lookupEnv", stub.ExpectArgs{"DISPLAY"}, nil, nil),
		}, nil, nil, &hst.AppError{
			Step: "finalise",
			Err:  os.ErrInvalid,
			Msg:  "DISPLAY is not set",
		}, nil, nil, nil, nil, nil},

		{"abs stat", func(bool, bool) outcomeOp {
			return new(spX11Op)
		}, func() *hst.Config {
			c := hst.Template()
			*c.Enablements |= hst.Enablements(hst.EX11)
			return c
		}, nil, []stub.Call{
			call("lookupEnv", stub.ExpectArgs{"DISPLAY"}, "unix:/tmp/.X11-unix/X0", nil),
			call("stat", stub.ExpectArgs{"/tmp/.X11-unix/X0"}, (*stubFi)(nil), stub.UniqueError(0)),
		}, nil, nil, &hst.AppError{
			Step: `access X11 socket "/tmp/.X11-unix/X0"`,
			Err:  stub.UniqueError(0),
		}, nil, nil, nil, nil, nil},

		{"success abs nonexistent", func(isShim, _ bool) outcomeOp {
			if !isShim {
				return new(spX11Op)
			}
			return &spX11Op{Display: "unix:/tmp/.X11-unix/X0"}
		}, func() *hst.Config {
			c := hst.Template()
			*c.Enablements |= hst.Enablements(hst.EX11)
			return c
		}, nil, []stub.Call{
			call("lookupEnv", stub.ExpectArgs{"DISPLAY"}, "unix:/tmp/.X11-unix/X0", nil),
			call("stat", stub.ExpectArgs{"/tmp/.X11-unix/X0"}, (*stubFi)(nil), os.ErrNotExist),
		}, newI().
			ChangeHosts("#1000009"), nil, nil, insertsOps(nil), []stub.Call{
			// this op configures the container state and does not make calls during toContainer
		}, &container.Params{
			Ops: new(container.Ops).
				Bind(absX11SocketDir, absX11SocketDir, 0),
		}, paramsWantEnv(config, map[string]string{
			"DISPLAY": "unix:/tmp/.X11-unix/X0",
		}, nil), nil},

		{"success abs abstract", func(isShim, _ bool) outcomeOp {
			if !isShim {
				return new(spX11Op)
			}
			return &spX11Op{Display: "unix:/tmp/.X11-unix/X0"}
		}, func() *hst.Config {
			c := hst.Template()
			*c.Enablements |= hst.Enablements(hst.EX11)
			c.Container.Flags &= ^hst.FHostAbstract
			return c
		}, nil, []stub.Call{
			call("lookupEnv", stub.ExpectArgs{"DISPLAY"}, "unix:/tmp/.X11-unix/X0", nil),
			call("stat", stub.ExpectArgs{"/tmp/.X11-unix/X0"}, (*stubFi)(nil), nil),
		}, newI().
			UpdatePermType(hst.EX11, m("/tmp/.X11-unix/X0"), acl.Read, acl.Write, acl.Execute).
			ChangeHosts("#1000009"), nil, nil, insertsOps(nil), []stub.Call{
			// this op configures the container state and does not make calls during toContainer
		}, &container.Params{
			Ops: new(container.Ops).
				Bind(absX11SocketDir, absX11SocketDir, 0),
		}, paramsWantEnv(config, map[string]string{
			"DISPLAY": "unix:/tmp/.X11-unix/X0",
		}, nil), nil},

		{"success", func(isShim, _ bool) outcomeOp {
			if !isShim {
				return new(spX11Op)
			}
			return &spX11Op{Display: ":0"}
		}, func() *hst.Config {
			c := hst.Template()
			*c.Enablements |= hst.Enablements(hst.EX11)
			return c
		}, nil, []stub.Call{
			call("lookupEnv", stub.ExpectArgs{"DISPLAY"}, ":0", nil),
			call("stat", stub.ExpectArgs{"/tmp/.X11-unix/X0"}, (*stubFi)(nil), nil),
		}, newI().
			UpdatePermType(hst.EX11, m("/tmp/.X11-unix/X0"), acl.Read, acl.Write, acl.Execute).
			ChangeHosts("#1000009"), nil, nil, insertsOps(nil), []stub.Call{
			// this op configures the container state and does not make calls during toContainer
		}, &container.Params{
			Ops: new(container.Ops).
				Bind(absX11SocketDir, absX11SocketDir, 0),
		}, paramsWantEnv(config, map[string]string{
			"DISPLAY": ":0",
		}, nil), nil},
	})
}
