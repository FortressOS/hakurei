package app

import (
	"os"
	"syscall"
	"testing"

	"hakurei.app/container"
	"hakurei.app/container/stub"
	"hakurei.app/hst"
)

func TestSpAccountOp(t *testing.T) {
	t.Parallel()
	config := hst.Template()

	checkOpBehaviour(t, []opBehaviourTestCase{
		{"invalid state", func(bool, bool) outcomeOp { return spAccountOp{} }, func() *hst.Config {
			c := hst.Template()
			c.Container.Shell = nil
			return c
		}, nil, []stub.Call{
			// this op performs basic validation and does not make calls during toSystem
		}, nil, nil, syscall.ENOTRECOVERABLE, nil, nil, nil, nil, nil},

		{"invalid user name", func(bool, bool) outcomeOp { return spAccountOp{} }, func() *hst.Config {
			c := hst.Template()
			c.Container.Username = "9"
			return c
		}, nil, []stub.Call{
			// this op performs basic validation and does not make calls during toSystem
		}, nil, nil, &hst.AppError{
			Step: "finalise",
			Err:  os.ErrInvalid,
			Msg:  `invalid user name "9"`,
		}, nil, nil, nil, nil, nil},

		{"success fallback username", func(bool, bool) outcomeOp { return spAccountOp{} }, func() *hst.Config {
			c := hst.Template()
			c.Container.Username = ""
			return c
		}, nil, []stub.Call{
			// this op performs basic validation and does not make calls during toSystem
		}, newI(), nil, nil, insertsOps(nil), []stub.Call{
			// this op configures the container state and does not make calls during toContainer
		}, &container.Params{
			Dir: config.Container.Home,
			Ops: new(container.Ops).
				Place(m("/etc/passwd"), []byte("chronos:x:1000:100:Hakurei:/data/data/org.chromium.Chromium:/run/current-system/sw/bin/zsh\n")).
				Place(m("/etc/group"), []byte("hakurei:x:100:\n")),
		}, paramsWantEnv(config, map[string]string{
			"HOME":  config.Container.Home.String(),
			"USER":  config.Container.Username,
			"SHELL": config.Container.Shell.String(),
		}, nil), nil},

		{"success", func(bool, bool) outcomeOp { return spAccountOp{} }, hst.Template, nil, []stub.Call{
			// this op performs basic validation and does not make calls during toSystem
		}, newI(), nil, nil, insertsOps(nil), []stub.Call{
			// this op configures the container state and does not make calls during toContainer
		}, &container.Params{
			Dir: config.Container.Home,
			Ops: new(container.Ops).
				Place(m("/etc/passwd"), []byte("chronos:x:1000:100:Hakurei:/data/data/org.chromium.Chromium:/run/current-system/sw/bin/zsh\n")).
				Place(m("/etc/group"), []byte("hakurei:x:100:\n")),
		}, paramsWantEnv(config, map[string]string{
			"HOME":  config.Container.Home.String(),
			"USER":  config.Container.Username,
			"SHELL": config.Container.Shell.String(),
		}, nil), nil},
	})
}
