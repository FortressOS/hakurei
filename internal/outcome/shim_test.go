package outcome

import (
	"bytes"
	"context"
	"log"
	"os"
	"syscall"
	"testing"

	"hakurei.app/container"
	"hakurei.app/container/comp"
	"hakurei.app/container/fhs"
	"hakurei.app/container/seccomp"
	"hakurei.app/container/stub"
	"hakurei.app/hst"
	"hakurei.app/internal/env"
)

func TestShimEntrypoint(t *testing.T) {
	t.Parallel()
	shimPreset := seccomp.Preset(comp.PresetStrict, seccomp.AllowMultiarch)
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
			Root(m("/var/lib/hakurei/base/org.debian"), comp.BindWritable).
			// spParamsOp
			Proc(fhs.AbsProc).
			Tmpfs(hst.AbsPrivateTmp, 1<<12, 0755).
			Bind(fhs.AbsDev, fhs.AbsDev, comp.BindWritable|comp.BindDevice).
			Tmpfs(fhs.AbsDev.Append("shm"), 0, 01777).

			// spRuntimeOp
			Tmpfs(fhs.AbsRunUser, 1<<12, 0755).
			Bind(m("/tmp/hakurei.10/runtime/9999"), m("/run/user/1000"), comp.BindWritable).

			// spTmpdirOp
			Bind(m("/tmp/hakurei.10/tmpdir/9999"), fhs.AbsTmp, comp.BindWritable).

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
				comp.BindWritable|comp.BindEnsure).
			Bind(fhs.AbsDev.Append("dri"), fhs.AbsDev.Append("dri"),
				comp.BindOptional|comp.BindWritable|comp.BindDevice).
			Remount(fhs.AbsRoot, syscall.MS_RDONLY),
	}

	checkSimple(t, "shimEntrypoint", []simpleTestCase{
		{"success", func(k *kstub) error { shimEntrypoint(k); return nil }, stub.Expect{Calls: []stub.Call{
			call("getMsg", stub.ExpectArgs{}, nil, nil),
			call("getLogger", stub.ExpectArgs{}, (*log.Logger)(nil), nil),
			call("setDumpable", stub.ExpectArgs{uintptr(container.SUID_DUMP_DISABLE)}, nil, nil),
			call("receive", stub.ExpectArgs{"HAKUREI_SHIM", outcomeState{
				Shim: &shimParams{PrivPID: 0xbad, WaitDelay: 0xf, Verbose: true, Ops: []outcomeOp{
					&spParamsOp{"xterm-256color", true},
					&spRuntimeOp{sessionTypeWayland},
					spTmpdirOp{},
					spAccountOp{},
					&spWaylandOp{},
					&spPulseOp{(*[pulseCookieSizeMax]byte)(bytes.Repeat([]byte{0}, pulseCookieSizeMax)), pulseCookieSizeMax},
					&spDBusOp{true},
					&spFilesystemOp{},
				}},

				ID:        &checkExpectInstanceId,
				Identity:  hst.IdentityMax,
				UserID:    10,
				Container: hst.Template().Container,
				Mapuid:    1000,
				Mapgid:    100,
				Paths:     &env.Paths{TempDir: fhs.AbsTmp, RuntimePath: fhs.AbsRunUser.Append("1000")},
			}, nil}, nil, nil),
			call("swapVerbose", stub.ExpectArgs{true}, false, nil),
			call("verbosef", stub.ExpectArgs{"process share directory at %q, runtime directory at %q", []any{m("/tmp/hakurei.10"), m("/run/user/1000/hakurei")}}, nil, nil),
			call("setupContSignal", stub.ExpectArgs{0xbad}, 0, nil),
			call("prctl", stub.ExpectArgs{uintptr(syscall.PR_SET_PDEATHSIG), uintptr(syscall.SIGCONT), uintptr(0)}, nil, nil),
			call("New", stub.ExpectArgs{}, nil, nil),
			call("closeReceive", stub.ExpectArgs{}, nil, nil),
			call("notifyContext", stub.ExpectArgs{context.Background(), []os.Signal{os.Interrupt, syscall.SIGTERM}}, nil, nil),
			call("containerStart", stub.ExpectArgs{templateParams}, nil, nil),
			call("containerServe", stub.ExpectArgs{templateParams}, nil, nil),
			call("seccompLoad", stub.ExpectArgs{shimPreset, seccomp.AllowMultiarch}, nil, nil),
			call("containerWait", stub.ExpectArgs{templateParams}, nil, nil),

			// deferred
			call("wKeepAlive", stub.ExpectArgs{}, nil, nil),
		}, Tracks: []stub.Expect{{Calls: []stub.Call{
			call("rcRead", stub.ExpectArgs{}, []byte{2}, nil),
			call("verbose", stub.ExpectArgs{[]any{"sa_sigaction got invalid siginfo"}}, nil, nil),
			call("rcRead", stub.ExpectArgs{}, []byte{3}, nil),
			call("verbose", stub.ExpectArgs{[]any{"got SIGCONT from unexpected process"}}, nil, nil),
			call("rcRead", stub.ExpectArgs{}, nil, nil), // stub terminates this goroutine
		}}}}, nil},
	})
}
