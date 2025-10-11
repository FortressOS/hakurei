package app

import (
	"maps"
	"os"
	"reflect"
	"testing"

	"hakurei.app/container"
	"hakurei.app/container/bits"
	"hakurei.app/container/fhs"
	"hakurei.app/container/seccomp"
	"hakurei.app/container/stub"
	"hakurei.app/hst"
)

func TestSpParamsOp(t *testing.T) {
	config := hst.Template()

	checkOpBehaviour(t, []opBehaviourTestCase{
		{"invalid program path", func(isShim, _ bool) outcomeOp {
			if !isShim {
				return new(spParamsOp)
			}
			return &spParamsOp{Term: "xterm", TermSet: true}
		}, func() *hst.Config {
			c := hst.Template()
			c.Container.Path = nil
			return c
		}, nil, []stub.Call{
			call("lookupEnv", stub.ExpectArgs{"TERM"}, "xterm", nil),
		}, newI().
			Ensure(m(container.Nonexistent+"/tmp/hakurei.0"), 0711), nil, nil, nil, []stub.Call{
			// this op configures the container state and does not make calls during toContainer
		}, nil, nil, &hst.AppError{
			Step: "finalise",
			Err:  os.ErrInvalid,
			Msg:  "invalid program path",
		}},

		{"success defaultargs secure", func(isShim, _ bool) outcomeOp {
			if !isShim {
				return new(spParamsOp)
			}
			return &spParamsOp{Term: "xterm", TermSet: true}
		}, func() *hst.Config {
			c := hst.Template()
			c.Container.Args = nil
			c.Container.Multiarch = false
			c.Container.SeccompCompat = false
			c.Container.Devel = false
			c.Container.Userns = false
			c.Container.Tty = false
			c.Container.Device = false
			return c
		}, nil, []stub.Call{
			call("lookupEnv", stub.ExpectArgs{"TERM"}, "xterm", nil),
		}, newI().
			Ensure(m(container.Nonexistent+"/tmp/hakurei.0"), 0711), nil, nil, nil, []stub.Call{
			// this op configures the container state and does not make calls during toContainer
		}, &container.Params{
			Hostname:       config.Container.Hostname,
			HostNet:        config.Container.HostNet,
			HostAbstract:   config.Container.HostAbstract,
			Path:           config.Container.Path,
			Args:           []string{config.Container.Path.String()},
			SeccompPresets: bits.PresetExt | bits.PresetDenyDevel | bits.PresetDenyNS | bits.PresetDenyTTY,
			Uid:            1000,
			Gid:            100,
			Ops: new(container.Ops).
				Root(m("/var/lib/hakurei/base/org.debian"), bits.BindWritable).
				Proc(fhs.AbsProc).Tmpfs(hst.AbsPrivateTmp, 1<<12, 0755).
				DevWritable(fhs.AbsDev, true).
				Tmpfs(fhs.AbsDev.Append("shm"), 0, 01777),
		}, func(t *testing.T, state *outcomeStateParams) {
			wantEnv := map[string]string{
				"TERM": "xterm",
			}
			maps.Copy(wantEnv, config.Container.Env)
			if !maps.Equal(state.env, wantEnv) {
				t.Errorf("toContainer: env = %#v, want %#v", state.env, wantEnv)
			}

			const wantAutoEtcPrefix = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			if state.as.AutoEtcPrefix != wantAutoEtcPrefix {
				t.Errorf("toContainer: as.AutoEtcPrefix = %q, want %q", state.as.AutoEtcPrefix, wantAutoEtcPrefix)
			}

			wantFilesystems := config.Container.Filesystem[1:]
			if !reflect.DeepEqual(state.filesystem, wantFilesystems) {
				t.Errorf("toContainer: filesystem = %#v, want %#v", state.filesystem, wantFilesystems)
			}
		}, nil},

		{"success", func(isShim, _ bool) outcomeOp {
			if !isShim {
				return new(spParamsOp)
			}
			return &spParamsOp{Term: "xterm", TermSet: true}
		}, hst.Template, nil, []stub.Call{
			call("lookupEnv", stub.ExpectArgs{"TERM"}, "xterm", nil),
		}, newI().
			Ensure(m(container.Nonexistent+"/tmp/hakurei.0"), 0711), nil, nil, nil, []stub.Call{
			// this op configures the container state and does not make calls during toContainer
		}, &container.Params{
			Hostname:      config.Container.Hostname,
			RetainSession: config.Container.Tty,
			HostNet:       config.Container.HostNet,
			HostAbstract:  config.Container.HostAbstract,
			Path:          config.Container.Path,
			Args:          config.Container.Args,
			SeccompFlags:  seccomp.AllowMultiarch,
			Uid:           1000,
			Gid:           100,
			Ops: new(container.Ops).
				Root(m("/var/lib/hakurei/base/org.debian"), bits.BindWritable).
				Proc(fhs.AbsProc).Tmpfs(hst.AbsPrivateTmp, 1<<12, 0755).
				Bind(fhs.AbsDev, fhs.AbsDev, bits.BindWritable|bits.BindDevice).
				Tmpfs(fhs.AbsDev.Append("shm"), 0, 01777),
		}, func(t *testing.T, state *outcomeStateParams) {
			wantEnv := map[string]string{
				"TERM": "xterm",
			}
			maps.Copy(wantEnv, config.Container.Env)
			if !maps.Equal(state.env, wantEnv) {
				t.Errorf("toContainer: env = %#v, want %#v", state.env, wantEnv)
			}

			const wantAutoEtcPrefix = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			if state.as.AutoEtcPrefix != wantAutoEtcPrefix {
				t.Errorf("toContainer: as.AutoEtcPrefix = %q, want %q", state.as.AutoEtcPrefix, wantAutoEtcPrefix)
			}

			wantFilesystems := config.Container.Filesystem[1:]
			if !reflect.DeepEqual(state.filesystem, wantFilesystems) {
				t.Errorf("toContainer: filesystem = %#v, want %#v", state.filesystem, wantFilesystems)
			}
		}, nil},
	})
}
