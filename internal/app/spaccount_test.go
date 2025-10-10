package app

import (
	"maps"
	"os"
	"syscall"
	"testing"

	"hakurei.app/container"
	"hakurei.app/container/stub"
	"hakurei.app/hst"
)

func TestSpAccountOp(t *testing.T) {
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
		}, newI(), nil, nil, func(state *outcomeStateParams) {
			state.params.Ops = new(container.Ops)
		}, []stub.Call{
			// this op configures the container state and does not make calls during toContainer
		}, &container.Params{
			Dir: config.Container.Home,
			Ops: new(container.Ops).
				Place(m("/etc/passwd"), []byte("chronos:x:1000:100:Hakurei:/data/data/org.chromium.Chromium:/run/current-system/sw/bin/zsh\n")).
				Place(m("/etc/group"), []byte("hakurei:x:100:\n")),
		}, func(t *testing.T, state *outcomeStateParams) {
			wantEnv := map[string]string{
				"HOME":  config.Container.Home.String(),
				"USER":  config.Container.Username,
				"SHELL": config.Container.Shell.String(),
			}
			maps.Copy(wantEnv, config.Container.Env)
			if !maps.Equal(state.env, wantEnv) {
				t.Errorf("toContainer: env = %#v, want %#v", state.env, wantEnv)
			}
		}, nil},

		{"success", func(bool, bool) outcomeOp { return spAccountOp{} }, hst.Template, nil, []stub.Call{
			// this op performs basic validation and does not make calls during toSystem
		}, newI(), nil, nil, func(state *outcomeStateParams) {
			state.params.Ops = new(container.Ops)
		}, []stub.Call{
			// this op configures the container state and does not make calls during toContainer
		}, &container.Params{
			Dir: config.Container.Home,
			Ops: new(container.Ops).
				Place(m("/etc/passwd"), []byte("chronos:x:1000:100:Hakurei:/data/data/org.chromium.Chromium:/run/current-system/sw/bin/zsh\n")).
				Place(m("/etc/group"), []byte("hakurei:x:100:\n")),
		}, func(t *testing.T, state *outcomeStateParams) {
			wantEnv := map[string]string{
				"HOME":  config.Container.Home.String(),
				"USER":  config.Container.Username,
				"SHELL": config.Container.Shell.String(),
			}
			maps.Copy(wantEnv, config.Container.Env)
			if !maps.Equal(state.env, wantEnv) {
				t.Errorf("toContainer: env = %#v, want %#v", state.env, wantEnv)
			}
		}, nil},
	})
}
