package app

import (
	"bytes"
	"os"
	"testing"

	"hakurei.app/container"
	"hakurei.app/container/check"
	"hakurei.app/container/stub"
	"hakurei.app/hst"
	"hakurei.app/system"
	"hakurei.app/system/acl"
)

func TestSpPulseOp(t *testing.T) {
	t.Parallel()

	config := hst.Template()
	sampleCookie := bytes.Repeat([]byte{0xfc}, pulseCookieSizeMax)

	checkOpBehaviour(t, []opBehaviourTestCase{
		{"not enabled", func(bool, bool) outcomeOp {
			return new(spPulseOp)
		}, func() *hst.Config {
			c := hst.Template()
			*c.Enablements = 0
			return c
		}, nil, nil, nil, nil, errNotEnabled, nil, nil, nil, nil, nil},

		{"socketDir stat", func(isShim bool, _ bool) outcomeOp {
			if !isShim {
				return new(spPulseOp)
			}
			return &spPulseOp{Cookie: (*[256]byte)(sampleCookie)}
		}, hst.Template, nil, []stub.Call{
			call("stat", stub.ExpectArgs{wantRuntimePath + "/pulse"}, (*stubFi)(nil), stub.UniqueError(2)),
		}, nil, nil, &hst.AppError{
			Step: `access PulseAudio directory "/proc/nonexistent/xdg_runtime_dir/pulse"`,
			Err:  stub.UniqueError(2),
		}, nil, nil, nil, nil, nil},

		{"socketDir nonexistent", func(bool, bool) outcomeOp {
			return new(spPulseOp)
		}, hst.Template, nil, []stub.Call{
			call("stat", stub.ExpectArgs{wantRuntimePath + "/pulse"}, (*stubFi)(nil), os.ErrNotExist),
		}, nil, nil, &hst.AppError{
			Step: "finalise",
			Err:  os.ErrNotExist,
			Msg:  `PulseAudio directory "/proc/nonexistent/xdg_runtime_dir/pulse" not found`,
		}, nil, nil, nil, nil, nil},

		{"socket stat", func(bool, bool) outcomeOp {
			return new(spPulseOp)
		}, hst.Template, nil, []stub.Call{
			call("stat", stub.ExpectArgs{wantRuntimePath + "/pulse"}, (*stubFi)(nil), nil),
			call("stat", stub.ExpectArgs{wantRuntimePath + "/pulse/native"}, (*stubFi)(nil), stub.UniqueError(1)),
		}, nil, nil, &hst.AppError{
			Step: `access PulseAudio socket "/proc/nonexistent/xdg_runtime_dir/pulse/native"`,
			Err:  stub.UniqueError(1),
		}, nil, nil, nil, nil, nil},

		{"socket nonexistent", func(bool, bool) outcomeOp {
			return new(spPulseOp)
		}, hst.Template, nil, []stub.Call{
			call("stat", stub.ExpectArgs{wantRuntimePath + "/pulse"}, (*stubFi)(nil), nil),
			call("stat", stub.ExpectArgs{wantRuntimePath + "/pulse/native"}, (*stubFi)(nil), os.ErrNotExist),
		}, nil, nil, &hst.AppError{
			Step: "finalise",
			Err:  os.ErrNotExist,
			Msg:  `PulseAudio directory "/proc/nonexistent/xdg_runtime_dir/pulse" found but socket does not exist`,
		}, nil, nil, nil, nil, nil},

		{"socket mode", func(bool, bool) outcomeOp {
			return new(spPulseOp)
		}, hst.Template, nil, []stub.Call{
			call("stat", stub.ExpectArgs{wantRuntimePath + "/pulse"}, (*stubFi)(nil), nil),
			call("stat", stub.ExpectArgs{wantRuntimePath + "/pulse/native"}, &stubFi{mode: 0660}, nil),
		}, nil, nil, &hst.AppError{
			Step: "finalise",
			Err:  os.ErrInvalid,
			Msg:  `unexpected permissions on "/proc/nonexistent/xdg_runtime_dir/pulse/native": -rw-rw----`,
		}, nil, nil, nil, nil, nil},

		{"cookie notAbs", func(bool, bool) outcomeOp {
			return new(spPulseOp)
		}, hst.Template, nil, []stub.Call{
			call("stat", stub.ExpectArgs{wantRuntimePath + "/pulse"}, (*stubFi)(nil), nil),
			call("stat", stub.ExpectArgs{wantRuntimePath + "/pulse/native"}, &stubFi{mode: 0666}, nil),
			call("lookupEnv", stub.ExpectArgs{"PULSE_COOKIE"}, "proc/nonexistent/cookie", nil),
		}, nil, nil, &hst.AppError{
			Step: "locate PulseAudio cookie",
			Err:  &check.AbsoluteError{Pathname: "proc/nonexistent/cookie"},
		}, nil, nil, nil, nil, nil},

		{"cookie loadFile", func(bool, bool) outcomeOp {
			return new(spPulseOp)
		}, hst.Template, nil, []stub.Call{
			call("stat", stub.ExpectArgs{wantRuntimePath + "/pulse"}, (*stubFi)(nil), nil),
			call("stat", stub.ExpectArgs{wantRuntimePath + "/pulse/native"}, &stubFi{mode: 0666}, nil),
			call("lookupEnv", stub.ExpectArgs{"PULSE_COOKIE"}, "/proc/nonexistent/cookie", nil),
			call("verbosef", stub.ExpectArgs{"loading up to %d bytes from %q", []any{1 << 8, "/proc/nonexistent/cookie"}}, nil, nil),
			call("stat", stub.ExpectArgs{"/proc/nonexistent/cookie"}, &stubFi{isDir: false, size: 1 << 8}, nil),
			call("open", stub.ExpectArgs{"/proc/nonexistent/cookie"}, (*stubOsFile)(nil), stub.UniqueError(0)),
		}, nil, nil, &hst.AppError{
			Step: "open PulseAudio cookie",
			Err:  stub.UniqueError(0),
		}, nil, nil, nil, nil, nil},

		{"success cookie", func(isShim bool, _ bool) outcomeOp {
			if !isShim {
				return new(spPulseOp)
			}
			return &spPulseOp{Cookie: (*[256]byte)(sampleCookie)}
		}, hst.Template, nil, []stub.Call{
			call("stat", stub.ExpectArgs{wantRuntimePath + "/pulse"}, (*stubFi)(nil), nil),
			call("stat", stub.ExpectArgs{wantRuntimePath + "/pulse/native"}, &stubFi{mode: 0666}, nil),
			call("lookupEnv", stub.ExpectArgs{"PULSE_COOKIE"}, "/proc/nonexistent/cookie", nil),
			call("verbosef", stub.ExpectArgs{"loading up to %d bytes from %q", []any{1 << 8, "/proc/nonexistent/cookie"}}, nil, nil),
			call("stat", stub.ExpectArgs{"/proc/nonexistent/cookie"}, &stubFi{isDir: false, size: 1 << 8}, nil),
			call("open", stub.ExpectArgs{"/proc/nonexistent/cookie"}, &stubOsFile{Reader: bytes.NewReader(sampleCookie)}, nil),
		}, newI().
			// state.ensureRuntimeDir
			Ensure(m(wantRunDirPath), 0700).
			UpdatePermType(system.User, m(wantRunDirPath), acl.Execute).
			Ensure(m(wantRuntimePath), 0700).
			UpdatePermType(system.User, m(wantRuntimePath), acl.Execute).
			// state.runtime
			Ephemeral(system.Process, m(wantRuntimeSharePath), 0700).
			UpdatePerm(m(wantRuntimeSharePath), acl.Execute).
			// toSystem
			Link(m(wantRuntimePath+"/pulse/native"), m(wantRuntimeSharePath+"/pulse")), sysUsesRuntime(nil), nil, insertsOps(afterSpRuntimeOp(nil)), []stub.Call{
			// this op configures the container state and does not make calls during toContainer
		}, &container.Params{
			Ops: new(container.Ops).
				Bind(m(wantRuntimeSharePath+"/pulse"), m("/run/user/1000/pulse/native"), 0).
				Place(m("/.hakurei/pulse-cookie"), sampleCookie),
		}, paramsWantEnv(config, map[string]string{
			"PULSE_SERVER": "unix:/run/user/1000/pulse/native",
			"PULSE_COOKIE": "/.hakurei/pulse-cookie",
		}, nil), nil},

		{"success", func(bool, bool) outcomeOp {
			return new(spPulseOp)
		}, hst.Template, nil, []stub.Call{
			call("stat", stub.ExpectArgs{wantRuntimePath + "/pulse"}, (*stubFi)(nil), nil),
			call("stat", stub.ExpectArgs{wantRuntimePath + "/pulse/native"}, &stubFi{mode: 0666}, nil),
			call("lookupEnv", stub.ExpectArgs{"PULSE_COOKIE"}, nil, nil),
			call("lookupEnv", stub.ExpectArgs{"HOME"}, nil, nil),
			call("lookupEnv", stub.ExpectArgs{"XDG_CONFIG_HOME"}, nil, nil),
			call("verbose", stub.ExpectArgs{[]any{"cannot locate PulseAudio cookie (tried $PULSE_COOKIE, $XDG_CONFIG_HOME/pulse/cookie, $HOME/.pulse-cookie)"}}, nil, nil),
		}, newI().
			// state.ensureRuntimeDir
			Ensure(m(wantRunDirPath), 0700).
			UpdatePermType(system.User, m(wantRunDirPath), acl.Execute).
			Ensure(m(wantRuntimePath), 0700).
			UpdatePermType(system.User, m(wantRuntimePath), acl.Execute).
			// state.runtime
			Ephemeral(system.Process, m(wantRuntimeSharePath), 0700).
			UpdatePerm(m(wantRuntimeSharePath), acl.Execute).
			// toSystem
			Link(m(wantRuntimePath+"/pulse/native"), m(wantRuntimeSharePath+"/pulse")), sysUsesRuntime(nil), nil, insertsOps(afterSpRuntimeOp(nil)), []stub.Call{
			// this op configures the container state and does not make calls during toContainer
		}, &container.Params{
			Ops: new(container.Ops).
				Bind(m(wantRuntimeSharePath+"/pulse"), m("/run/user/1000/pulse/native"), 0),
		}, paramsWantEnv(config, map[string]string{
			"PULSE_SERVER": "unix:/run/user/1000/pulse/native",
		}, nil), nil},
	})
}
