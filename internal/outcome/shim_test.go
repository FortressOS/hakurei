package outcome

import (
	"bytes"
	"context"
	"io"
	"log"
	"os"
	"syscall"
	"testing"

	"hakurei.app/container"
	"hakurei.app/container/fhs"
	"hakurei.app/container/seccomp"
	"hakurei.app/container/std"
	"hakurei.app/container/stub"
	"hakurei.app/hst"
	"hakurei.app/internal/env"
)

func TestShimEntrypoint(t *testing.T) {
	t.Parallel()
	shimPreset := seccomp.Preset(std.PresetStrict, seccomp.AllowMultiarch)
	templateParams := &container.Params{
		Dir: m("/data/data/org.chromium.Chromium"),
		Env: []string{
			"DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus",
			"DBUS_SYSTEM_BUS_ADDRESS=unix:path=/var/run/dbus/system_bus_socket",
			"GOOGLE_API_KEY=AIzaSyBHDrl33hwRp4rMQY0ziRbj8K9LPA6vUCY",
			"GOOGLE_DEFAULT_CLIENT_ID=77185425430.apps.googleusercontent.com",
			"GOOGLE_DEFAULT_CLIENT_SECRET=OTJgUOQcT7lO7GsGZq2G4IlT",
			"HOME=/data/data/org.chromium.Chromium",
			"PULSE_COOKIE=/.hakurei/pulse-cookie",
			"PULSE_SERVER=unix:/run/user/1000/pulse/native",
			"SHELL=/run/current-system/sw/bin/zsh",
			"TERM=xterm-256color",
			"USER=chronos",
			"WAYLAND_DISPLAY=wayland-0",
			"XDG_RUNTIME_DIR=/run/user/1000",
			"XDG_SESSION_CLASS=user",
			"XDG_SESSION_TYPE=wayland",
		},

		// spParamsOp
		Hostname:      "localhost",
		RetainSession: true,
		HostNet:       true,
		HostAbstract:  true,
		ForwardCancel: true,
		Path:          m("/run/current-system/sw/bin/chromium"),
		Args: []string{
			"chromium",
			"--ignore-gpu-blocklist",
			"--disable-smooth-scrolling",
			"--enable-features=UseOzonePlatform",
			"--ozone-platform=wayland",
		},
		SeccompFlags: seccomp.AllowMultiarch,
		Uid:          1000,
		Gid:          100,

		Ops: new(container.Ops).
			// resolveRoot
			Root(m("/var/lib/hakurei/base/org.debian"), std.BindWritable).
			// spParamsOp
			Proc(fhs.AbsProc).
			Tmpfs(hst.AbsPrivateTmp, 1<<12, 0755).
			Bind(fhs.AbsDev, fhs.AbsDev, std.BindWritable|std.BindDevice).
			Tmpfs(fhs.AbsDev.Append("shm"), 0, 01777).

			// spRuntimeOp
			Tmpfs(fhs.AbsRunUser, 1<<12, 0755).
			Bind(m("/tmp/hakurei.10/runtime/9999"), m("/run/user/1000"), std.BindWritable).

			// spTmpdirOp
			Bind(m("/tmp/hakurei.10/tmpdir/9999"), fhs.AbsTmp, std.BindWritable).

			// spAccountOp
			Place(m("/etc/passwd"), []byte("chronos:x:1000:100:Hakurei:/data/data/org.chromium.Chromium:/run/current-system/sw/bin/zsh\n")).
			Place(m("/etc/group"), []byte("hakurei:x:100:\n")).

			// spWaylandOp
			Bind(m("/tmp/hakurei.10/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/wayland"), m("/run/user/1000/wayland-0"), 0).

			// spPulseOp
			Bind(m("/run/user/1000/hakurei/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/pulse"), m("/run/user/1000/pulse/native"), 0).
			Place(m("/.hakurei/pulse-cookie"), bytes.Repeat([]byte{0}, pulseCookieSizeMax)).

			// spDBusOp
			Bind(m("/tmp/hakurei.10/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/bus"), m("/run/user/1000/bus"), 0).
			Bind(m("/tmp/hakurei.10/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/system_bus_socket"), m("/var/run/dbus/system_bus_socket"), 0).

			// spFilesystemOp
			Etc(fhs.AbsEtc, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa").
			Tmpfs(fhs.AbsTmp, 0, 0755).
			Overlay(m("/nix/store"),
				fhs.AbsVarLib.Append("hakurei/nix/u0/org.chromium.Chromium/rw-store/upper"),
				fhs.AbsVarLib.Append("hakurei/nix/u0/org.chromium.Chromium/rw-store/work"),
				fhs.AbsVarLib.Append("hakurei/base/org.nixos/ro-store")).
			Link(m("/run/current-system"), "/run/current-system", true).
			Link(m("/run/opengl-driver"), "/run/opengl-driver", true).
			Bind(fhs.AbsVarLib.Append("hakurei/u0/org.chromium.Chromium"),
				m("/data/data/org.chromium.Chromium"),
				std.BindWritable|std.BindEnsure).
			Bind(fhs.AbsDev.Append("dri"), fhs.AbsDev.Append("dri"),
				std.BindOptional|std.BindWritable|std.BindDevice).
			Remount(fhs.AbsRoot, syscall.MS_RDONLY),
	}

	newShimParams := func() *shimParams {
		return &shimParams{PrivPID: 0xbad, WaitDelay: 0xf, Verbose: true, Ops: []outcomeOp{
			&spParamsOp{"xterm-256color", true},
			&spRuntimeOp{sessionTypeWayland},
			spTmpdirOp{},
			spAccountOp{},
			&spWaylandOp{},
			&spPulseOp{(*[pulseCookieSizeMax]byte)(bytes.Repeat([]byte{0}, pulseCookieSizeMax)), pulseCookieSizeMax},
			&spDBusOp{true},
			&spFilesystemOp{},
		}}
	}

	templateState := outcomeState{
		Shim:      newShimParams(),
		ID:        &checkExpectInstanceId,
		Identity:  hst.IdentityEnd,
		UserID:    10,
		Container: hst.Template().Container,
		Mapuid:    1000,
		Mapgid:    100,
		Paths:     &env.Paths{TempDir: fhs.AbsTmp, RuntimePath: fhs.AbsRunUser.Append("1000")},
	}

	checkSimple(t, "shimEntrypoint", []simpleTestCase{
		{"dumpable", func(k *kstub) error { shimEntrypoint(k); return nil }, stub.Expect{Calls: []stub.Call{
			call("getMsg", stub.ExpectArgs{}, nil, nil),
			call("getLogger", stub.ExpectArgs{}, new(log.Logger), nil),
			call("setDumpable", stub.ExpectArgs{uintptr(container.SUID_DUMP_DISABLE)}, nil, stub.UniqueError(11)),
			call("fatalf", stub.ExpectArgs{"cannot set SUID_DUMP_DISABLE: %v", []any{stub.UniqueError(11)}}, nil, nil),
		}}, nil},

		{"receive exit request", func(k *kstub) error { shimEntrypoint(k); return nil }, stub.Expect{Calls: []stub.Call{
			call("getMsg", stub.ExpectArgs{}, nil, nil),
			call("getLogger", stub.ExpectArgs{}, (*log.Logger)(nil), nil),
			call("setDumpable", stub.ExpectArgs{uintptr(container.SUID_DUMP_DISABLE)}, nil, nil),
			call("getppid", stub.ExpectArgs{}, 0xbad, nil),
			call("setupContSignal", stub.ExpectArgs{0xbad}, 0, nil),
			call("receive", stub.ExpectArgs{"HAKUREI_SHIM", outcomeState{}, nil}, nil, io.EOF),
			call("exit", stub.ExpectArgs{hst.ExitRequest}, stub.PanicExit, nil),

			// deferred
			call("wKeepAlive", stub.ExpectArgs{}, nil, nil),
		}}, nil},

		{"receive fd", func(k *kstub) error { shimEntrypoint(k); return nil }, stub.Expect{Calls: []stub.Call{
			call("getMsg", stub.ExpectArgs{}, nil, nil),
			call("getLogger", stub.ExpectArgs{}, (*log.Logger)(nil), nil),
			call("setDumpable", stub.ExpectArgs{uintptr(container.SUID_DUMP_DISABLE)}, nil, nil),
			call("getppid", stub.ExpectArgs{}, 0xbad, nil),
			call("setupContSignal", stub.ExpectArgs{0xbad}, 0, nil),
			call("receive", stub.ExpectArgs{"HAKUREI_SHIM", outcomeState{}, nil}, nil, syscall.EBADF),
			call("fatal", stub.ExpectArgs{[]any{"invalid config descriptor"}}, nil, nil),

			// deferred
			call("wKeepAlive", stub.ExpectArgs{}, nil, nil),
		}}, nil},

		{"receive env", func(k *kstub) error { shimEntrypoint(k); return nil }, stub.Expect{Calls: []stub.Call{
			call("getMsg", stub.ExpectArgs{}, nil, nil),
			call("getLogger", stub.ExpectArgs{}, (*log.Logger)(nil), nil),
			call("setDumpable", stub.ExpectArgs{uintptr(container.SUID_DUMP_DISABLE)}, nil, nil),
			call("getppid", stub.ExpectArgs{}, 0xbad, nil),
			call("setupContSignal", stub.ExpectArgs{0xbad}, 0, nil),
			call("receive", stub.ExpectArgs{"HAKUREI_SHIM", outcomeState{}, nil}, nil, container.ErrReceiveEnv),
			call("fatal", stub.ExpectArgs{[]any{"HAKUREI_SHIM not set"}}, nil, nil),

			// deferred
			call("wKeepAlive", stub.ExpectArgs{}, nil, nil),
		}}, nil},

		{"receive strange", func(k *kstub) error { shimEntrypoint(k); return nil }, stub.Expect{Calls: []stub.Call{
			call("getMsg", stub.ExpectArgs{}, nil, nil),
			call("getLogger", stub.ExpectArgs{}, (*log.Logger)(nil), nil),
			call("setDumpable", stub.ExpectArgs{uintptr(container.SUID_DUMP_DISABLE)}, nil, nil),
			call("getppid", stub.ExpectArgs{}, 0xbad, nil),
			call("setupContSignal", stub.ExpectArgs{0xbad}, 0, nil),
			call("receive", stub.ExpectArgs{"HAKUREI_SHIM", outcomeState{}, nil}, nil, stub.UniqueError(10)),
			call("fatalf", stub.ExpectArgs{"cannot receive shim setup params: %v", []any{stub.UniqueError(10)}}, nil, nil),

			// deferred
			call("wKeepAlive", stub.ExpectArgs{}, nil, nil),
		}}, nil},

		{"reparent", func(k *kstub) error { shimEntrypoint(k); return nil }, stub.Expect{Calls: []stub.Call{
			call("getMsg", stub.ExpectArgs{}, nil, nil),
			call("getLogger", stub.ExpectArgs{}, (*log.Logger)(nil), nil),
			call("setDumpable", stub.ExpectArgs{uintptr(container.SUID_DUMP_DISABLE)}, nil, nil),
			call("getppid", stub.ExpectArgs{}, 0xbad, nil),
			call("setupContSignal", stub.ExpectArgs{0xbad}, 0, nil),
			call("receive", stub.ExpectArgs{"HAKUREI_SHIM", func() outcomeState {
				state := templateState
				state.Shim = newShimParams()
				state.Shim.PrivPID = 0xfff
				return state
			}(), nil}, nil, nil),
			call("swapVerbose", stub.ExpectArgs{true}, false, nil),
			call("verbosef", stub.ExpectArgs{"process share directory at %q, runtime directory at %q", []any{m("/tmp/hakurei.10"), m("/run/user/1000/hakurei")}}, nil, nil),
			call("fatalf", stub.ExpectArgs{"unexpectedly reparented from %d to %d", []any{0xfff, 0xbad}}, nil, nil),

			// deferred
			call("wKeepAlive", stub.ExpectArgs{}, nil, nil),
		}}, nil},

		{"invalid state", func(k *kstub) error { shimEntrypoint(k); return nil }, stub.Expect{Calls: []stub.Call{
			call("getMsg", stub.ExpectArgs{}, nil, nil),
			call("getLogger", stub.ExpectArgs{}, (*log.Logger)(nil), nil),
			call("setDumpable", stub.ExpectArgs{uintptr(container.SUID_DUMP_DISABLE)}, nil, nil),
			call("getppid", stub.ExpectArgs{}, 0xbad, nil),
			call("setupContSignal", stub.ExpectArgs{0xbad}, 0, nil),
			call("receive", stub.ExpectArgs{"HAKUREI_SHIM", func() outcomeState {
				state := templateState
				state.Shim = newShimParams()
				state.Shim.PrivPID = 0
				return state
			}(), nil}, nil, nil),
			call("swapVerbose", stub.ExpectArgs{true}, false, nil),
			call("fatal", stub.ExpectArgs{[]any{"impossible outcome state reached\n"}}, nil, nil),

			// deferred
			call("wKeepAlive", stub.ExpectArgs{}, nil, nil),
		}}, nil},

		{"sigaction pipe", func(k *kstub) error { shimEntrypoint(k); return nil }, stub.Expect{Calls: []stub.Call{
			call("getMsg", stub.ExpectArgs{}, nil, nil),
			call("getLogger", stub.ExpectArgs{}, (*log.Logger)(nil), nil),
			call("setDumpable", stub.ExpectArgs{uintptr(container.SUID_DUMP_DISABLE)}, nil, nil),
			call("getppid", stub.ExpectArgs{}, 0xbad, nil),
			call("setupContSignal", stub.ExpectArgs{0xbad}, 0, &os.SyscallError{Syscall: "pipe2", Err: stub.UniqueError(9)}),
			call("fatal", stub.ExpectArgs{[]any{"pipe2: unique error 9 injected by the test suite"}}, nil, nil),
		}}, nil},

		{"sigaction cgo", func(k *kstub) error { shimEntrypoint(k); return nil }, stub.Expect{Calls: []stub.Call{
			call("getMsg", stub.ExpectArgs{}, nil, nil),
			call("getLogger", stub.ExpectArgs{}, (*log.Logger)(nil), nil),
			call("setDumpable", stub.ExpectArgs{uintptr(container.SUID_DUMP_DISABLE)}, nil, nil),
			call("getppid", stub.ExpectArgs{}, 0xbad, nil),
			call("setupContSignal", stub.ExpectArgs{0xbad}, 0, syscall.ENOTRECOVERABLE),
			call("fatalf", stub.ExpectArgs{"cannot install SIGCONT handler: %v", []any{syscall.ENOTRECOVERABLE}}, nil, nil),
		}}, nil},

		{"sigaction strange", func(k *kstub) error { shimEntrypoint(k); return nil }, stub.Expect{Calls: []stub.Call{
			call("getMsg", stub.ExpectArgs{}, nil, nil),
			call("getLogger", stub.ExpectArgs{}, (*log.Logger)(nil), nil),
			call("setDumpable", stub.ExpectArgs{uintptr(container.SUID_DUMP_DISABLE)}, nil, nil),
			call("getppid", stub.ExpectArgs{}, 0xbad, nil),
			call("setupContSignal", stub.ExpectArgs{0xbad}, 0, stub.UniqueError(8)),
			call("fatalf", stub.ExpectArgs{"cannot set up exit request: %v", []any{stub.UniqueError(8)}}, nil, nil),
		}}, nil},

		{"prctl", func(k *kstub) error { shimEntrypoint(k); return nil }, stub.Expect{Calls: []stub.Call{
			call("getMsg", stub.ExpectArgs{}, nil, nil),
			call("getLogger", stub.ExpectArgs{}, (*log.Logger)(nil), nil),
			call("setDumpable", stub.ExpectArgs{uintptr(container.SUID_DUMP_DISABLE)}, nil, nil),
			call("getppid", stub.ExpectArgs{}, 0xbad, nil),
			call("setupContSignal", stub.ExpectArgs{0xbad}, 0, nil),
			call("receive", stub.ExpectArgs{"HAKUREI_SHIM", templateState, nil}, nil, nil),
			call("swapVerbose", stub.ExpectArgs{true}, false, nil),
			call("verbosef", stub.ExpectArgs{"process share directory at %q, runtime directory at %q", []any{m("/tmp/hakurei.10"), m("/run/user/1000/hakurei")}}, nil, nil),
			call("prctl", stub.ExpectArgs{uintptr(syscall.PR_SET_PDEATHSIG), uintptr(syscall.SIGCONT), uintptr(0)}, nil, stub.UniqueError(7)),
			call("fatalf", stub.ExpectArgs{"cannot set parent-death signal: %v", []any{stub.UniqueError(7)}}, nil, nil),

			// deferred
			call("wKeepAlive", stub.ExpectArgs{}, nil, nil),
		}}, nil},

		{"toContainer", func(k *kstub) error { shimEntrypoint(k); return nil }, stub.Expect{Calls: []stub.Call{
			call("getMsg", stub.ExpectArgs{}, nil, nil),
			call("getLogger", stub.ExpectArgs{}, (*log.Logger)(nil), nil),
			call("setDumpable", stub.ExpectArgs{uintptr(container.SUID_DUMP_DISABLE)}, nil, nil),
			call("getppid", stub.ExpectArgs{}, 0xbad, nil),
			call("setupContSignal", stub.ExpectArgs{0xbad}, 0, nil),
			call("receive", stub.ExpectArgs{"HAKUREI_SHIM", func() outcomeState {
				state := templateState
				state.Shim = newShimParams()
				state.Shim.Ops = []outcomeOp{errorOp(6)}
				return state
			}(), nil}, nil, nil),
			call("swapVerbose", stub.ExpectArgs{true}, false, nil),
			call("verbosef", stub.ExpectArgs{"process share directory at %q, runtime directory at %q", []any{m("/tmp/hakurei.10"), m("/run/user/1000/hakurei")}}, nil, nil),
			call("prctl", stub.ExpectArgs{uintptr(syscall.PR_SET_PDEATHSIG), uintptr(syscall.SIGCONT), uintptr(0)}, nil, nil),
			call("fatal", stub.ExpectArgs{[]any{"cannot create container state: unique error 6 injected by the test suite\n"}}, nil, nil),

			// deferred
			call("wKeepAlive", stub.ExpectArgs{}, nil, nil),
		}}, nil},

		{"bad ops", func(k *kstub) error { shimEntrypoint(k); return nil }, stub.Expect{Calls: []stub.Call{
			call("getMsg", stub.ExpectArgs{}, nil, nil),
			call("getLogger", stub.ExpectArgs{}, (*log.Logger)(nil), nil),
			call("setDumpable", stub.ExpectArgs{uintptr(container.SUID_DUMP_DISABLE)}, nil, nil),
			call("getppid", stub.ExpectArgs{}, 0xbad, nil),
			call("setupContSignal", stub.ExpectArgs{0xbad}, 0, nil),
			call("receive", stub.ExpectArgs{"HAKUREI_SHIM", func() outcomeState {
				state := templateState
				state.Shim = newShimParams()
				state.Shim.Ops = nil
				return state
			}(), nil}, nil, nil),
			call("swapVerbose", stub.ExpectArgs{true}, false, nil),
			call("verbosef", stub.ExpectArgs{"process share directory at %q, runtime directory at %q", []any{m("/tmp/hakurei.10"), m("/run/user/1000/hakurei")}}, nil, nil),
			call("prctl", stub.ExpectArgs{uintptr(syscall.PR_SET_PDEATHSIG), uintptr(syscall.SIGCONT), uintptr(0)}, nil, nil),
			call("fatal", stub.ExpectArgs{[]any{"invalid container state"}}, nil, nil),

			// deferred
			call("wKeepAlive", stub.ExpectArgs{}, nil, nil),
		}}, nil},

		{"start", func(k *kstub) error { shimEntrypoint(k); return nil }, stub.Expect{Calls: []stub.Call{
			call("getMsg", stub.ExpectArgs{}, nil, nil),
			call("getLogger", stub.ExpectArgs{}, (*log.Logger)(nil), nil),
			call("setDumpable", stub.ExpectArgs{uintptr(container.SUID_DUMP_DISABLE)}, nil, nil),
			call("getppid", stub.ExpectArgs{}, 0xbad, nil),
			call("setupContSignal", stub.ExpectArgs{0xbad}, 0, nil),
			call("receive", stub.ExpectArgs{"HAKUREI_SHIM", templateState, nil}, nil, nil),
			call("swapVerbose", stub.ExpectArgs{true}, false, nil),
			call("verbosef", stub.ExpectArgs{"process share directory at %q, runtime directory at %q", []any{m("/tmp/hakurei.10"), m("/run/user/1000/hakurei")}}, nil, nil),
			call("prctl", stub.ExpectArgs{uintptr(syscall.PR_SET_PDEATHSIG), uintptr(syscall.SIGCONT), uintptr(0)}, nil, nil),
			call("New", stub.ExpectArgs{}, nil, nil),
			call("closeReceive", stub.ExpectArgs{}, nil, nil),
			call("notifyContext", stub.ExpectArgs{context.Background(), []os.Signal{os.Interrupt, syscall.SIGTERM}}, -1, nil),
			call("containerStart", stub.ExpectArgs{templateParams}, nil, stub.UniqueError(5)),
			call("getLogger", stub.ExpectArgs{}, (*log.Logger)(nil), nil),
			call("verbose", stub.ExpectArgs{[]any{"cannot start container: unique error 5 injected by the test suite\n"}}, nil, nil),
			call("exit", stub.ExpectArgs{hst.ExitFailure}, stub.PanicExit, nil),

			// deferred
			call("wKeepAlive", stub.ExpectArgs{}, nil, nil),
		}, Tracks: []stub.Expect{{Calls: []stub.Call{
			call("rcRead", stub.ExpectArgs{}, nil, nil), // stub terminates this goroutine
		}}}}, nil},

		{"start logger signalread", func(k *kstub) error { shimEntrypoint(k); return nil }, stub.Expect{Calls: []stub.Call{
			call("getMsg", stub.ExpectArgs{}, nil, nil),
			call("getLogger", stub.ExpectArgs{}, (*log.Logger)(nil), nil),
			call("setDumpable", stub.ExpectArgs{uintptr(container.SUID_DUMP_DISABLE)}, nil, nil),
			call("getppid", stub.ExpectArgs{}, 0xbad, nil),
			call("setupContSignal", stub.ExpectArgs{0xbad}, 0, nil),
			call("receive", stub.ExpectArgs{"HAKUREI_SHIM", templateState, nil}, nil, nil),
			call("swapVerbose", stub.ExpectArgs{true}, false, nil),
			call("verbosef", stub.ExpectArgs{"process share directory at %q, runtime directory at %q", []any{m("/tmp/hakurei.10"), m("/run/user/1000/hakurei")}}, nil, nil),
			call("prctl", stub.ExpectArgs{uintptr(syscall.PR_SET_PDEATHSIG), uintptr(syscall.SIGCONT), uintptr(0)}, nil, nil),
			call("New", stub.ExpectArgs{}, nil, nil),
			call("closeReceive", stub.ExpectArgs{}, nil, nil),
			call("notifyContext", stub.ExpectArgs{context.Background(), []os.Signal{os.Interrupt, syscall.SIGTERM}}, -1, nil),
			call("containerStart", stub.ExpectArgs{templateParams}, nil, stub.UniqueError(5)),
			call("getLogger", stub.ExpectArgs{}, log.Default(), nil),
			call("exit", stub.ExpectArgs{hst.ExitFailure}, stub.PanicExit, nil),

			// deferred
			call("wKeepAlive", stub.ExpectArgs{}, nil, nil),
		}, Tracks: []stub.Expect{{Calls: []stub.Call{
			call("rcRead", stub.ExpectArgs{}, []byte{}, stub.UniqueError(4)),
			call("fatalf", stub.ExpectArgs{"cannot read from signal pipe: %v", []any{stub.UniqueError(4)}}, nil, nil),
		}}}}, nil},

		{"serve", func(k *kstub) error { shimEntrypoint(k); return nil }, stub.Expect{Calls: []stub.Call{
			call("getMsg", stub.ExpectArgs{}, nil, nil),
			call("getLogger", stub.ExpectArgs{}, (*log.Logger)(nil), nil),
			call("setDumpable", stub.ExpectArgs{uintptr(container.SUID_DUMP_DISABLE)}, nil, nil),
			call("getppid", stub.ExpectArgs{}, 0xbad, nil),
			call("setupContSignal", stub.ExpectArgs{0xbad}, 0, nil),
			call("receive", stub.ExpectArgs{"HAKUREI_SHIM", templateState, nil}, nil, nil),
			call("swapVerbose", stub.ExpectArgs{true}, false, nil),
			call("verbosef", stub.ExpectArgs{"process share directory at %q, runtime directory at %q", []any{m("/tmp/hakurei.10"), m("/run/user/1000/hakurei")}}, nil, nil),
			call("prctl", stub.ExpectArgs{uintptr(syscall.PR_SET_PDEATHSIG), uintptr(syscall.SIGCONT), uintptr(0)}, nil, nil),
			call("New", stub.ExpectArgs{}, nil, nil),
			call("closeReceive", stub.ExpectArgs{}, nil, nil),
			call("notifyContext", stub.ExpectArgs{context.Background(), []os.Signal{os.Interrupt, syscall.SIGTERM}}, -1, nil),
			call("containerStart", stub.ExpectArgs{templateParams}, nil, nil),
			call("containerServe", stub.ExpectArgs{templateParams}, nil, stub.UniqueError(3)),
			call("fatal", stub.ExpectArgs{[]any{"cannot configure container: unique error 3 injected by the test suite\n"}}, nil, nil),

			// deferred
			call("wKeepAlive", stub.ExpectArgs{}, nil, nil),
		}, Tracks: []stub.Expect{{Calls: []stub.Call{
			call("rcRead", stub.ExpectArgs{}, nil, nil), // stub terminates this goroutine
		}}}}, nil},

		{"seccomp", func(k *kstub) error { shimEntrypoint(k); return nil }, stub.Expect{Calls: []stub.Call{
			call("getMsg", stub.ExpectArgs{}, nil, nil),
			call("getLogger", stub.ExpectArgs{}, (*log.Logger)(nil), nil),
			call("setDumpable", stub.ExpectArgs{uintptr(container.SUID_DUMP_DISABLE)}, nil, nil),
			call("getppid", stub.ExpectArgs{}, 0xbad, nil),
			call("setupContSignal", stub.ExpectArgs{0xbad}, 0, nil),
			call("receive", stub.ExpectArgs{"HAKUREI_SHIM", templateState, nil}, nil, nil),
			call("swapVerbose", stub.ExpectArgs{true}, false, nil),
			call("verbosef", stub.ExpectArgs{"process share directory at %q, runtime directory at %q", []any{m("/tmp/hakurei.10"), m("/run/user/1000/hakurei")}}, nil, nil),
			call("prctl", stub.ExpectArgs{uintptr(syscall.PR_SET_PDEATHSIG), uintptr(syscall.SIGCONT), uintptr(0)}, nil, nil),
			call("New", stub.ExpectArgs{}, nil, nil),
			call("closeReceive", stub.ExpectArgs{}, nil, nil),
			call("notifyContext", stub.ExpectArgs{context.Background(), []os.Signal{os.Interrupt, syscall.SIGTERM}}, -1, nil),
			call("containerStart", stub.ExpectArgs{templateParams}, nil, nil),
			call("containerServe", stub.ExpectArgs{templateParams}, nil, nil),
			call("seccompLoad", stub.ExpectArgs{shimPreset, seccomp.AllowMultiarch}, nil, stub.UniqueError(2)),
			call("fatalf", stub.ExpectArgs{"cannot load syscall filter: %v", []any{stub.UniqueError(2)}}, nil, nil),

			// deferred
			call("wKeepAlive", stub.ExpectArgs{}, nil, nil),
		}, Tracks: []stub.Expect{{Calls: []stub.Call{
			call("rcRead", stub.ExpectArgs{}, nil, nil), // stub terminates this goroutine
		}}}}, nil},

		{"exited closesetup earlyrequested", func(k *kstub) error { shimEntrypoint(k); return nil }, stub.Expect{Calls: []stub.Call{
			call("getMsg", stub.ExpectArgs{}, nil, nil),
			call("getLogger", stub.ExpectArgs{}, (*log.Logger)(nil), nil),
			call("setDumpable", stub.ExpectArgs{uintptr(container.SUID_DUMP_DISABLE)}, nil, nil),
			call("getppid", stub.ExpectArgs{}, 0xbad, nil),
			call("setupContSignal", stub.ExpectArgs{0xbad}, 0, nil),
			call("receive", stub.ExpectArgs{"HAKUREI_SHIM", templateState, nil}, nil, nil),
			call("swapVerbose", stub.ExpectArgs{true}, false, nil),
			call("verbosef", stub.ExpectArgs{"process share directory at %q, runtime directory at %q", []any{m("/tmp/hakurei.10"), m("/run/user/1000/hakurei")}}, nil, nil),
			call("prctl", stub.ExpectArgs{uintptr(syscall.PR_SET_PDEATHSIG), uintptr(syscall.SIGCONT), uintptr(0)}, nil, nil),
			call("New", stub.ExpectArgs{}, nil, nil),
			call("closeReceive", stub.ExpectArgs{}, nil, stub.UniqueError(1)),
			call("verbosef", stub.ExpectArgs{"cannot close setup pipe: %v", []any{stub.UniqueError(1)}}, nil, nil),
			call("notifyContext", stub.ExpectArgs{context.Background(), []os.Signal{os.Interrupt, syscall.SIGTERM}}, 0, nil),
			call("containerStart", stub.ExpectArgs{templateParams}, nil, nil),
			call("containerServe", stub.ExpectArgs{templateParams}, nil, nil),
			call("seccompLoad", stub.ExpectArgs{shimPreset, seccomp.AllowMultiarch}, nil, nil),
			call("containerWait", stub.ExpectArgs{templateParams}, nil, makeExitError(1<<8)),
			call("exit", stub.ExpectArgs{1}, stub.PanicExit, nil),

			// deferred
			call("wKeepAlive", stub.ExpectArgs{}, nil, nil),
		}, Tracks: []stub.Expect{{Calls: []stub.Call{
			call("rcRead", stub.ExpectArgs{}, []byte{shimMsgExitRequested}, nil),
			call("exit", stub.ExpectArgs{hst.ExitRequest}, stub.PanicExit, unblockNotifyContext),
		}}}}, nil},

		{"exited requested", func(k *kstub) error { shimEntrypoint(k); return nil }, stub.Expect{Calls: []stub.Call{
			call("getMsg", stub.ExpectArgs{}, nil, nil),
			call("getLogger", stub.ExpectArgs{}, (*log.Logger)(nil), nil),
			call("setDumpable", stub.ExpectArgs{uintptr(container.SUID_DUMP_DISABLE)}, nil, nil),
			call("getppid", stub.ExpectArgs{}, 0xbad, nil),
			call("setupContSignal", stub.ExpectArgs{0xbad}, 0, nil),
			call("receive", stub.ExpectArgs{"HAKUREI_SHIM", templateState, nil}, nil, nil),
			call("swapVerbose", stub.ExpectArgs{true}, false, nil),
			call("verbosef", stub.ExpectArgs{"process share directory at %q, runtime directory at %q", []any{m("/tmp/hakurei.10"), m("/run/user/1000/hakurei")}}, nil, nil),
			call("prctl", stub.ExpectArgs{uintptr(syscall.PR_SET_PDEATHSIG), uintptr(syscall.SIGCONT), uintptr(0)}, nil, nil),
			call("New", stub.ExpectArgs{}, nil, nil),
			call("closeReceive", stub.ExpectArgs{}, nil, nil),
			call("notifyContext", stub.ExpectArgs{context.Background(), []os.Signal{os.Interrupt, syscall.SIGTERM}}, 0, nil),
			call("containerStart", stub.ExpectArgs{templateParams}, nil, nil),
			call("containerServe", stub.ExpectArgs{templateParams}, nil, nil),
			call("seccompLoad", stub.ExpectArgs{shimPreset, seccomp.AllowMultiarch}, nil, nil),
			call("containerWait", stub.ExpectArgs{templateParams}, nil, makeExitError(1<<8)),
			call("exit", stub.ExpectArgs{1}, stub.PanicExit, nil),

			// deferred
			call("wKeepAlive", stub.ExpectArgs{}, nil, nil),
		}, Tracks: []stub.Expect{{Calls: []stub.Call{
			call("rcRead", stub.ExpectArgs{}, []byte{shimMsgExitRequested}, unblockNotifyContext),
			call("notifyContextStop", stub.ExpectArgs{}, nil, nil),
			call("rcRead", stub.ExpectArgs{}, nil, nil), // stub terminates this goroutine
		}}}}, nil},

		{"canceled orphaned", func(k *kstub) error { shimEntrypoint(k); return nil }, stub.Expect{Calls: []stub.Call{
			call("getMsg", stub.ExpectArgs{}, nil, nil),
			call("getLogger", stub.ExpectArgs{}, (*log.Logger)(nil), nil),
			call("setDumpable", stub.ExpectArgs{uintptr(container.SUID_DUMP_DISABLE)}, nil, nil),
			call("getppid", stub.ExpectArgs{}, 0xbad, nil),
			call("setupContSignal", stub.ExpectArgs{0xbad}, 0, nil),
			call("receive", stub.ExpectArgs{"HAKUREI_SHIM", templateState, nil}, nil, nil),
			call("swapVerbose", stub.ExpectArgs{true}, false, nil),
			call("verbosef", stub.ExpectArgs{"process share directory at %q, runtime directory at %q", []any{m("/tmp/hakurei.10"), m("/run/user/1000/hakurei")}}, nil, nil),
			call("prctl", stub.ExpectArgs{uintptr(syscall.PR_SET_PDEATHSIG), uintptr(syscall.SIGCONT), uintptr(0)}, nil, nil),
			call("New", stub.ExpectArgs{}, nil, nil),
			call("closeReceive", stub.ExpectArgs{}, nil, nil),
			call("notifyContext", stub.ExpectArgs{context.Background(), []os.Signal{os.Interrupt, syscall.SIGTERM}}, -1, nil),
			call("containerStart", stub.ExpectArgs{templateParams}, nil, nil),
			call("containerServe", stub.ExpectArgs{templateParams}, nil, nil),
			call("seccompLoad", stub.ExpectArgs{shimPreset, seccomp.AllowMultiarch}, nil, nil),
			call("containerWait", stub.ExpectArgs{templateParams}, nil, context.Canceled),
			call("exit", stub.ExpectArgs{hst.ExitCancel}, stub.PanicExit, nil),

			// deferred
			call("wKeepAlive", stub.ExpectArgs{}, nil, nil),
		}, Tracks: []stub.Expect{{Calls: []stub.Call{
			call("rcRead", stub.ExpectArgs{}, []byte{shimMsgOrphaned}, nil),
			call("exit", stub.ExpectArgs{hst.ExitOrphan}, stub.PanicExit, nil),
		}}}}, nil},

		{"strangewait invalidmsg", func(k *kstub) error { shimEntrypoint(k); return nil }, stub.Expect{Calls: []stub.Call{
			call("getMsg", stub.ExpectArgs{}, nil, nil),
			call("getLogger", stub.ExpectArgs{}, (*log.Logger)(nil), nil),
			call("setDumpable", stub.ExpectArgs{uintptr(container.SUID_DUMP_DISABLE)}, nil, nil),
			call("getppid", stub.ExpectArgs{}, 0xbad, nil),
			call("setupContSignal", stub.ExpectArgs{0xbad}, 0, nil),
			call("receive", stub.ExpectArgs{"HAKUREI_SHIM", templateState, nil}, nil, nil),
			call("swapVerbose", stub.ExpectArgs{true}, false, nil),
			call("verbosef", stub.ExpectArgs{"process share directory at %q, runtime directory at %q", []any{m("/tmp/hakurei.10"), m("/run/user/1000/hakurei")}}, nil, nil),
			call("prctl", stub.ExpectArgs{uintptr(syscall.PR_SET_PDEATHSIG), uintptr(syscall.SIGCONT), uintptr(0)}, nil, nil),
			call("New", stub.ExpectArgs{}, nil, nil),
			call("closeReceive", stub.ExpectArgs{}, nil, nil),
			call("notifyContext", stub.ExpectArgs{context.Background(), []os.Signal{os.Interrupt, syscall.SIGTERM}}, -1, nil),
			call("containerStart", stub.ExpectArgs{templateParams}, nil, nil),
			call("containerServe", stub.ExpectArgs{templateParams}, nil, nil),
			call("seccompLoad", stub.ExpectArgs{shimPreset, seccomp.AllowMultiarch}, nil, nil),
			call("containerWait", stub.ExpectArgs{templateParams}, nil, stub.UniqueError(0)),
			call("verbosef", stub.ExpectArgs{"cannot wait: %v", []any{stub.UniqueError(0)}}, nil, nil),
			call("exit", stub.ExpectArgs{127}, stub.PanicExit, nil),

			// deferred
			call("wKeepAlive", stub.ExpectArgs{}, nil, nil),
		}, Tracks: []stub.Expect{{Calls: []stub.Call{
			call("rcRead", stub.ExpectArgs{}, []byte{0xff}, nil),
			call("fatalf", stub.ExpectArgs{"got invalid message %d from signal handler", []any{byte(0xff)}}, nil, nil),
		}}}}, nil},

		{"success", func(k *kstub) error { shimEntrypoint(k); return nil }, stub.Expect{Calls: []stub.Call{
			call("getMsg", stub.ExpectArgs{}, nil, nil),
			call("getLogger", stub.ExpectArgs{}, (*log.Logger)(nil), nil),
			call("setDumpable", stub.ExpectArgs{uintptr(container.SUID_DUMP_DISABLE)}, nil, nil),
			call("getppid", stub.ExpectArgs{}, 0xbad, nil),
			call("setupContSignal", stub.ExpectArgs{0xbad}, 0, nil),
			call("receive", stub.ExpectArgs{"HAKUREI_SHIM", templateState, nil}, nil, nil),
			call("swapVerbose", stub.ExpectArgs{true}, false, nil),
			call("verbosef", stub.ExpectArgs{"process share directory at %q, runtime directory at %q", []any{m("/tmp/hakurei.10"), m("/run/user/1000/hakurei")}}, nil, nil),
			call("prctl", stub.ExpectArgs{uintptr(syscall.PR_SET_PDEATHSIG), uintptr(syscall.SIGCONT), uintptr(0)}, nil, nil),
			call("New", stub.ExpectArgs{}, nil, nil),
			call("closeReceive", stub.ExpectArgs{}, nil, nil),
			call("notifyContext", stub.ExpectArgs{context.Background(), []os.Signal{os.Interrupt, syscall.SIGTERM}}, -1, nil),
			call("containerStart", stub.ExpectArgs{templateParams}, nil, nil),
			call("containerServe", stub.ExpectArgs{templateParams}, nil, nil),
			call("seccompLoad", stub.ExpectArgs{shimPreset, seccomp.AllowMultiarch}, nil, nil),
			call("containerWait", stub.ExpectArgs{templateParams}, nil, nil),

			// deferred
			call("wKeepAlive", stub.ExpectArgs{}, nil, nil),
		}, Tracks: []stub.Expect{{Calls: []stub.Call{
			call("rcRead", stub.ExpectArgs{}, []byte{shimMsgInvalid}, nil),
			call("verbose", stub.ExpectArgs{[]any{"sa_sigaction got invalid siginfo"}}, nil, nil),
			call("rcRead", stub.ExpectArgs{}, []byte{shimMsgBadPID}, nil),
			call("verbose", stub.ExpectArgs{[]any{"got SIGCONT from unexpected process"}}, nil, nil),
			call("rcRead", stub.ExpectArgs{}, nil, nil), // stub terminates this goroutine
		}}}}, nil},
	})
}

// errorOp implements a noop outcomeOp that unconditionally returns [stub.UniqueError].
type errorOp stub.UniqueError

func (e errorOp) toSystem(*outcomeStateSys) error       { return stub.UniqueError(e) }
func (e errorOp) toContainer(*outcomeStateParams) error { return stub.UniqueError(e) }
