package app

import (
	"bytes"
	"errors"
	"os"
	"syscall"
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

		{"socketDir stat", func(isShim, _ bool) outcomeOp {
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
			call("stat", stub.ExpectArgs{"/proc/nonexistent/cookie"}, &stubFi{isDir: false, size: 1 << 8}, nil),
			call("verbosef", stub.ExpectArgs{"loading %d bytes from %q", []any{1 << 8, "/proc/nonexistent/cookie"}}, nil, nil),
			call("open", stub.ExpectArgs{"/proc/nonexistent/cookie"}, (*stubOsFile)(nil), stub.UniqueError(0)),
		}, nil, nil, &hst.AppError{
			Step: "open PulseAudio cookie",
			Err:  stub.UniqueError(0),
		}, nil, nil, nil, nil, nil},

		{"cookie bad shim size", func(isShim, clearUnexported bool) outcomeOp {
			if !isShim {
				return new(spPulseOp)
			}
			op := &spPulseOp{Cookie: (*[pulseCookieSizeMax]byte)(sampleCookie), CookieSize: pulseCookieSizeMax}
			if clearUnexported {
				op.CookieSize += +0xfd
			}
			return op
		}, hst.Template, nil, []stub.Call{
			call("stat", stub.ExpectArgs{wantRuntimePath + "/pulse"}, (*stubFi)(nil), nil),
			call("stat", stub.ExpectArgs{wantRuntimePath + "/pulse/native"}, &stubFi{mode: 0666}, nil),
			call("lookupEnv", stub.ExpectArgs{"PULSE_COOKIE"}, "/proc/nonexistent/cookie", nil),
			call("stat", stub.ExpectArgs{"/proc/nonexistent/cookie"}, &stubFi{isDir: false, size: 1 << 8}, nil),
			call("verbosef", stub.ExpectArgs{"loading %d bytes from %q", []any{1 << 8, "/proc/nonexistent/cookie"}}, nil, nil),
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
		}, nil, nil, &hst.AppError{
			Step: "finalise",
			Err:  os.ErrInvalid,
			Msg:  "unexpected PulseAudio cookie size",
		}},

		{"success cookie short", func(isShim, _ bool) outcomeOp {
			if !isShim {
				return new(spPulseOp)
			}
			sampleCookieTrunc := make([]byte, pulseCookieSizeMax)
			copy(sampleCookieTrunc, sampleCookie[:len(sampleCookie)-0xe])
			return &spPulseOp{Cookie: (*[pulseCookieSizeMax]byte)(sampleCookieTrunc), CookieSize: pulseCookieSizeMax - 0xe}
		}, hst.Template, nil, []stub.Call{
			call("stat", stub.ExpectArgs{wantRuntimePath + "/pulse"}, (*stubFi)(nil), nil),
			call("stat", stub.ExpectArgs{wantRuntimePath + "/pulse/native"}, &stubFi{mode: 0666}, nil),
			call("lookupEnv", stub.ExpectArgs{"PULSE_COOKIE"}, "/proc/nonexistent/cookie", nil),
			call("stat", stub.ExpectArgs{"/proc/nonexistent/cookie"}, &stubFi{isDir: false, size: pulseCookieSizeMax - 0xe}, nil),
			call("verbosef", stub.ExpectArgs{"%s at %q is %d bytes shorter than expected", []any{"PulseAudio cookie", "/proc/nonexistent/cookie", int64(0xe)}}, nil, nil),
			call("open", stub.ExpectArgs{"/proc/nonexistent/cookie"}, &stubOsFile{Reader: bytes.NewReader(sampleCookie[:len(sampleCookie)-0xe])}, nil),
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
				Place(m("/.hakurei/pulse-cookie"), sampleCookie[:len(sampleCookie)-0xe]),
		}, paramsWantEnv(config, map[string]string{
			"PULSE_SERVER": "unix:/run/user/1000/pulse/native",
			"PULSE_COOKIE": "/.hakurei/pulse-cookie",
		}, nil), nil},

		{"success cookie", func(isShim, _ bool) outcomeOp {
			if !isShim {
				return new(spPulseOp)
			}
			return &spPulseOp{Cookie: (*[pulseCookieSizeMax]byte)(sampleCookie), CookieSize: pulseCookieSizeMax}
		}, hst.Template, nil, []stub.Call{
			call("stat", stub.ExpectArgs{wantRuntimePath + "/pulse"}, (*stubFi)(nil), nil),
			call("stat", stub.ExpectArgs{wantRuntimePath + "/pulse/native"}, &stubFi{mode: 0666}, nil),
			call("lookupEnv", stub.ExpectArgs{"PULSE_COOKIE"}, "/proc/nonexistent/cookie", nil),
			call("stat", stub.ExpectArgs{"/proc/nonexistent/cookie"}, &stubFi{isDir: false, size: 1 << 8}, nil),
			call("verbosef", stub.ExpectArgs{"loading %d bytes from %q", []any{1 << 8, "/proc/nonexistent/cookie"}}, nil, nil),
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

func TestDiscoverPulseCookie(t *testing.T) {
	t.Parallel()

	fCheckPathname := func(k *kstub) error {
		a, err := discoverPulseCookie(k)
		k.Verbose(a)
		return err
	}

	checkSimple(t, "discoverPulseCookie", []simpleTestCase{
		{"override notAbs", fCheckPathname, stub.Expect{Calls: []stub.Call{
			call("lookupEnv", stub.ExpectArgs{"PULSE_COOKIE"}, "proc/nonexistent/pulse-cookie", nil),
			call("verbose", stub.ExpectArgs{[]any{(*check.Absolute)(nil)}}, nil, nil),
		}}, &hst.AppError{
			Step: "locate PulseAudio cookie",
			Err:  &check.AbsoluteError{Pathname: "proc/nonexistent/pulse-cookie"},
		}},

		{"success override", fCheckPathname, stub.Expect{Calls: []stub.Call{
			call("lookupEnv", stub.ExpectArgs{"PULSE_COOKIE"}, "/proc/nonexistent/pulse-cookie", nil),
			call("verbose", stub.ExpectArgs{[]any{m("/proc/nonexistent/pulse-cookie")}}, nil, nil),
		}}, nil},

		{"home notAbs", fCheckPathname, stub.Expect{Calls: []stub.Call{
			call("lookupEnv", stub.ExpectArgs{"PULSE_COOKIE"}, nil, nil),
			call("lookupEnv", stub.ExpectArgs{"HOME"}, "proc/nonexistent/home", nil),
			call("verbose", stub.ExpectArgs{[]any{(*check.Absolute)(nil)}}, nil, nil),
		}}, &hst.AppError{
			Step: "locate PulseAudio cookie",
			Err:  &check.AbsoluteError{Pathname: "proc/nonexistent/home"},
		}},

		{"home stat", fCheckPathname, stub.Expect{Calls: []stub.Call{
			call("lookupEnv", stub.ExpectArgs{"PULSE_COOKIE"}, nil, nil),
			call("lookupEnv", stub.ExpectArgs{"HOME"}, "/proc/nonexistent/home", nil),
			call("stat", stub.ExpectArgs{"/proc/nonexistent/home/.pulse-cookie"}, (*stubFi)(nil), stub.UniqueError(1)),
			call("verbose", stub.ExpectArgs{[]any{(*check.Absolute)(nil)}}, nil, nil),
		}}, &hst.AppError{
			Step: "access PulseAudio cookie",
			Err:  stub.UniqueError(1),
		}},

		{"home nonexistent", fCheckPathname, stub.Expect{Calls: []stub.Call{
			call("lookupEnv", stub.ExpectArgs{"PULSE_COOKIE"}, nil, nil),
			call("lookupEnv", stub.ExpectArgs{"HOME"}, "/proc/nonexistent/home", nil),
			call("stat", stub.ExpectArgs{"/proc/nonexistent/home/.pulse-cookie"}, (*stubFi)(nil), os.ErrNotExist),
			call("lookupEnv", stub.ExpectArgs{"XDG_CONFIG_HOME"}, nil, nil),
			call("verbose", stub.ExpectArgs{[]any{(*check.Absolute)(nil)}}, nil, nil),
		}}, nil},

		{"success home", fCheckPathname, stub.Expect{Calls: []stub.Call{
			call("lookupEnv", stub.ExpectArgs{"PULSE_COOKIE"}, nil, nil),
			call("lookupEnv", stub.ExpectArgs{"HOME"}, "/proc/nonexistent/home", nil),
			call("stat", stub.ExpectArgs{"/proc/nonexistent/home/.pulse-cookie"}, &stubFi{}, nil),
			call("verbose", stub.ExpectArgs{[]any{m("/proc/nonexistent/home/.pulse-cookie")}}, nil, nil),
		}}, nil},

		{"xdg notAbs", fCheckPathname, stub.Expect{Calls: []stub.Call{
			call("lookupEnv", stub.ExpectArgs{"PULSE_COOKIE"}, nil, nil),
			call("lookupEnv", stub.ExpectArgs{"HOME"}, nil, nil),
			call("lookupEnv", stub.ExpectArgs{"XDG_CONFIG_HOME"}, "proc/nonexistent/xdg", nil),
			call("verbose", stub.ExpectArgs{[]any{(*check.Absolute)(nil)}}, nil, nil),
		}}, &hst.AppError{
			Step: "locate PulseAudio cookie",
			Err:  &check.AbsoluteError{Pathname: "proc/nonexistent/xdg"},
		}},

		{"xdg stat", fCheckPathname, stub.Expect{Calls: []stub.Call{
			call("lookupEnv", stub.ExpectArgs{"PULSE_COOKIE"}, nil, nil),
			call("lookupEnv", stub.ExpectArgs{"HOME"}, nil, nil),
			call("lookupEnv", stub.ExpectArgs{"XDG_CONFIG_HOME"}, "/proc/nonexistent/xdg", nil),
			call("stat", stub.ExpectArgs{"/proc/nonexistent/xdg/pulse/cookie"}, (*stubFi)(nil), stub.UniqueError(0)),
			call("verbose", stub.ExpectArgs{[]any{(*check.Absolute)(nil)}}, nil, nil),
		}}, &hst.AppError{
			Step: "access PulseAudio cookie",
			Err:  stub.UniqueError(0),
		}},

		{"xdg dir", fCheckPathname, stub.Expect{Calls: []stub.Call{
			call("lookupEnv", stub.ExpectArgs{"PULSE_COOKIE"}, nil, nil),
			call("lookupEnv", stub.ExpectArgs{"HOME"}, nil, nil),
			call("lookupEnv", stub.ExpectArgs{"XDG_CONFIG_HOME"}, "/proc/nonexistent/xdg", nil),
			call("stat", stub.ExpectArgs{"/proc/nonexistent/xdg/pulse/cookie"}, &stubFi{isDir: true}, nil),
			call("verbose", stub.ExpectArgs{[]any{(*check.Absolute)(nil)}}, nil, nil),
		}}, nil},

		{"success home dir xdg nonexistent", fCheckPathname, stub.Expect{Calls: []stub.Call{
			call("lookupEnv", stub.ExpectArgs{"PULSE_COOKIE"}, nil, nil),
			call("lookupEnv", stub.ExpectArgs{"HOME"}, "/proc/nonexistent/home", nil),
			call("stat", stub.ExpectArgs{"/proc/nonexistent/home/.pulse-cookie"}, &stubFi{isDir: true}, nil),
			call("lookupEnv", stub.ExpectArgs{"XDG_CONFIG_HOME"}, "/proc/nonexistent/xdg", nil),
			call("stat", stub.ExpectArgs{"/proc/nonexistent/xdg/pulse/cookie"}, (*stubFi)(nil), os.ErrNotExist),
			call("verbose", stub.ExpectArgs{[]any{(*check.Absolute)(nil)}}, nil, nil),
		}}, nil},

		{"success home nonexistent xdg", fCheckPathname, stub.Expect{Calls: []stub.Call{
			call("lookupEnv", stub.ExpectArgs{"PULSE_COOKIE"}, nil, nil),
			call("lookupEnv", stub.ExpectArgs{"HOME"}, "/proc/nonexistent/home", nil),
			call("stat", stub.ExpectArgs{"/proc/nonexistent/home/.pulse-cookie"}, (*stubFi)(nil), os.ErrNotExist),
			call("lookupEnv", stub.ExpectArgs{"XDG_CONFIG_HOME"}, "/proc/nonexistent/xdg", nil),
			call("stat", stub.ExpectArgs{"/proc/nonexistent/xdg/pulse/cookie"}, &stubFi{}, nil),
			call("verbose", stub.ExpectArgs{[]any{m("/proc/nonexistent/xdg/pulse/cookie")}}, nil, nil),
		}}, nil},

		{"success empty environ", fCheckPathname, stub.Expect{Calls: []stub.Call{
			call("lookupEnv", stub.ExpectArgs{"PULSE_COOKIE"}, nil, nil),
			call("lookupEnv", stub.ExpectArgs{"HOME"}, nil, nil),
			call("lookupEnv", stub.ExpectArgs{"XDG_CONFIG_HOME"}, nil, nil),
			call("verbose", stub.ExpectArgs{[]any{(*check.Absolute)(nil)}}, nil, nil),
		}}, nil},
	})
}

func TestLoadFile(t *testing.T) {
	t.Parallel()

	fAfterWriteExact := func(k *kstub) error {
		buf := make([]byte, 1<<8)
		n, err := loadFile(k, k,
			"simulated PulseAudio cookie",
			"/home/ophestra/xdg/config/pulse/cookie",
			buf)
		k.Verbose(buf[:n])
		return err
	}

	fAfterWrite := func(k *kstub) error {
		buf := make([]byte, 1<<8+0xfd)
		n, err := loadFile(k, k,
			"simulated PulseAudio cookie",
			"/home/ophestra/xdg/config/pulse/cookie",
			buf)
		k.Verbose(buf[:n])
		return err
	}

	fBeforeWrite := func(k *kstub) error {
		buf := make([]byte, 1<<8+0xfd)
		n, err := loadFile(k, k,
			"simulated PulseAudio cookie",
			"/home/ophestra/xdg/config/pulse/cookie",
			buf)
		k.Verbose(n)

		if !bytes.Equal(buf, make([]byte, len(buf))) {
			t.Errorf("loadFile: buf = %#v", buf)
		}
		return err
	}

	sampleCookie := bytes.Repeat([]byte{0xfc}, pulseCookieSizeMax)
	checkSimple(t, "loadFile", []simpleTestCase{
		{"buf", func(k *kstub) error {
			n, err := loadFile(k, k,
				"simulated PulseAudio cookie",
				"/home/ophestra/xdg/config/pulse/cookie",
				nil)
			k.Verbose(n)
			return err
		}, stub.Expect{Calls: []stub.Call{
			call("verbose", stub.ExpectArgs{[]any{-1}}, nil, nil),
		}}, errors.New("invalid buffer")},

		{"stat", fBeforeWrite, stub.Expect{Calls: []stub.Call{
			call("stat", stub.ExpectArgs{"/home/ophestra/xdg/config/pulse/cookie"}, (*stubFi)(nil), stub.UniqueError(3)),
			call("verbose", stub.ExpectArgs{[]any{-1}}, nil, nil),
		}}, &hst.AppError{
			Step: "access simulated PulseAudio cookie",
			Err:  stub.UniqueError(3),
		}},

		{"dir", fBeforeWrite, stub.Expect{Calls: []stub.Call{
			call("stat", stub.ExpectArgs{"/home/ophestra/xdg/config/pulse/cookie"}, &stubFi{isDir: true}, nil),
			call("verbose", stub.ExpectArgs{[]any{-1}}, nil, nil),
		}}, &hst.AppError{
			Step: "read simulated PulseAudio cookie",
			Err:  &os.PathError{Op: "stat", Path: "/home/ophestra/xdg/config/pulse/cookie", Err: syscall.EISDIR},
		}},

		{"oob", fBeforeWrite, stub.Expect{Calls: []stub.Call{
			call("stat", stub.ExpectArgs{"/home/ophestra/xdg/config/pulse/cookie"}, &stubFi{size: 1<<8 + 0xff}, nil),
			call("verbose", stub.ExpectArgs{[]any{-1}}, nil, nil),
		}}, &hst.AppError{
			Step: "finalise",
			Err:  &os.PathError{Op: "stat", Path: "/home/ophestra/xdg/config/pulse/cookie", Err: syscall.ENOMEM},
			Msg:  `simulated PulseAudio cookie at "/home/ophestra/xdg/config/pulse/cookie" exceeds expected size`,
		}},

		{"open", fBeforeWrite, stub.Expect{Calls: []stub.Call{
			call("stat", stub.ExpectArgs{"/home/ophestra/xdg/config/pulse/cookie"}, &stubFi{size: 1 << 8}, nil),
			call("verbosef", stub.ExpectArgs{"%s at %q is %d bytes shorter than expected", []any{"simulated PulseAudio cookie", "/home/ophestra/xdg/config/pulse/cookie", int64(0xfd)}}, nil, nil),
			call("open", stub.ExpectArgs{"/home/ophestra/xdg/config/pulse/cookie"}, (*stubOsFile)(nil), stub.UniqueError(2)),
			call("verbose", stub.ExpectArgs{[]any{-1}}, nil, nil),
		}}, &hst.AppError{Step: "open simulated PulseAudio cookie", Err: stub.UniqueError(2)}},

		{"read", fBeforeWrite, stub.Expect{Calls: []stub.Call{
			call("stat", stub.ExpectArgs{"/home/ophestra/xdg/config/pulse/cookie"}, &stubFi{size: 1 << 8}, nil),
			call("verbosef", stub.ExpectArgs{"%s at %q is %d bytes shorter than expected", []any{"simulated PulseAudio cookie", "/home/ophestra/xdg/config/pulse/cookie", int64(0xfd)}}, nil, nil),
			call("open", stub.ExpectArgs{"/home/ophestra/xdg/config/pulse/cookie"}, &stubOsFile{Reader: errorReader{stub.UniqueError(1)}}, nil),
			call("verbose", stub.ExpectArgs{[]any{-1}}, nil, nil),
		}}, &hst.AppError{Step: "read simulated PulseAudio cookie", Err: stub.UniqueError(1)}},

		{"short close", fAfterWrite, stub.Expect{Calls: []stub.Call{
			call("stat", stub.ExpectArgs{"/home/ophestra/xdg/config/pulse/cookie"}, &stubFi{size: 1 << 8}, nil),
			call("verbosef", stub.ExpectArgs{"%s at %q is %d bytes shorter than expected", []any{"simulated PulseAudio cookie", "/home/ophestra/xdg/config/pulse/cookie", int64(0xfd)}}, nil, nil),
			call("open", stub.ExpectArgs{"/home/ophestra/xdg/config/pulse/cookie"}, &stubOsFile{closeErr: stub.UniqueError(0), Reader: bytes.NewReader(sampleCookie)}, nil),
			call("verbose", stub.ExpectArgs{[]any{sampleCookie}}, nil, nil),
		}}, &hst.AppError{Step: "close simulated PulseAudio cookie", Err: stub.UniqueError(0)}},

		{"success", fAfterWriteExact, stub.Expect{Calls: []stub.Call{
			call("stat", stub.ExpectArgs{"/home/ophestra/xdg/config/pulse/cookie"}, &stubFi{size: 1 << 8}, nil),
			call("verbosef", stub.ExpectArgs{"loading %d bytes from %q", []any{1 << 8, "/home/ophestra/xdg/config/pulse/cookie"}}, nil, nil),
			call("open", stub.ExpectArgs{"/home/ophestra/xdg/config/pulse/cookie"}, &stubOsFile{Reader: bytes.NewReader(sampleCookie)}, nil),
			call("verbose", stub.ExpectArgs{[]any{sampleCookie}}, nil, nil),
		}}, nil},
	})
}
