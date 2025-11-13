package outcome

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os/exec"
	"os/user"
	"reflect"
	"syscall"
	"testing"
	"time"

	"hakurei.app/container"
	"hakurei.app/container/check"
	"hakurei.app/container/fhs"
	"hakurei.app/container/seccomp"
	"hakurei.app/container/std"
	"hakurei.app/hst"
	"hakurei.app/internal/system"
	"hakurei.app/internal/system/acl"
	"hakurei.app/internal/system/dbus"
	"hakurei.app/message"
)

func TestOutcomeMain(t *testing.T) {
	t.Parallel()
	msg := message.New(nil)
	msg.SwapVerbose(testing.Verbose())

	testCases := []struct {
		name       string
		k          syscallDispatcher
		config     *hst.Config
		id         hst.ID
		wantSys    *system.I
		wantParams *container.Params
	}{
		{"template", new(stubNixOS), hst.Template(), checkExpectInstanceId, system.New(panicMsgContext{}, message.New(nil), 10009).
			// spParamsOp
			Ensure(m("/tmp/hakurei.0"), 0711).

			// spRuntimeOp
			Ensure(m("/tmp/hakurei.0/runtime"), 0700).
			UpdatePermType(system.User, m("/tmp/hakurei.0/runtime"), acl.Execute).
			Ensure(m("/tmp/hakurei.0/runtime/9"), 0700).
			UpdatePermType(system.User, m("/tmp/hakurei.0/runtime/9"), acl.Read, acl.Write, acl.Execute).

			// spTmpdirOp
			Ensure(m("/tmp/hakurei.0/tmpdir"), 0700).
			UpdatePermType(system.User, m("/tmp/hakurei.0/tmpdir"), acl.Execute).
			Ensure(m("/tmp/hakurei.0/tmpdir/9"), 01700).
			UpdatePermType(system.User, m("/tmp/hakurei.0/tmpdir/9"), acl.Read, acl.Write, acl.Execute).

			// instance
			Ephemeral(system.Process, m("/tmp/hakurei.0/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), 0711).

			// spWaylandOp
			Wayland(
				m("/tmp/hakurei.0/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/wayland"),
				m("/run/user/1971/wayland-0"),
				"org.chromium.Chromium",
				"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			).

			// ensureRuntimeDir
			Ensure(m("/run/user/1971"), 0700).
			UpdatePermType(system.User, m("/run/user/1971"), acl.Execute).
			Ensure(m("/run/user/1971/hakurei"), 0700).
			UpdatePermType(system.User, m("/run/user/1971/hakurei"), acl.Execute).

			// runtime
			Ephemeral(system.Process, m("/run/user/1971/hakurei/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), 0700).
			UpdatePerm(m("/run/user/1971/hakurei/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), acl.Execute).

			// spPulseOp
			Link(m("/run/user/1971/pulse/native"), m("/run/user/1971/hakurei/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/pulse")).

			// spDBusOp
			MustProxyDBus(
				hst.Template().SessionBus,
				hst.Template().SystemBus, dbus.ProxyPair{
					"unix:path=/run/user/1971/bus",
					"/tmp/hakurei.0/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/bus",
				}, dbus.ProxyPair{
					"unix:path=/var/run/dbus/system_bus_socket",
					"/tmp/hakurei.0/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/system_bus_socket",
				},
			).UpdatePerm(m("/tmp/hakurei.0/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/bus"), acl.Read, acl.Write).
			UpdatePerm(m("/tmp/hakurei.0/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/system_bus_socket"), acl.Read, acl.Write).

			// spFilesystemOp
			Ensure(m("/var/lib/hakurei/u0"), 0700).
			UpdatePermType(system.User, m("/var/lib/hakurei/u0"), acl.Execute).
			UpdatePermType(system.User, m("/var/lib/hakurei/u0/org.chromium.Chromium"), acl.Read, acl.Write, acl.Execute), &container.Params{

			Dir: m("/data/data/org.chromium.Chromium"),
			Env: []string{
				"DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1971/bus",
				"DBUS_SYSTEM_BUS_ADDRESS=unix:path=/var/run/dbus/system_bus_socket",
				"GOOGLE_API_KEY=AIzaSyBHDrl33hwRp4rMQY0ziRbj8K9LPA6vUCY",
				"GOOGLE_DEFAULT_CLIENT_ID=77185425430.apps.googleusercontent.com",
				"GOOGLE_DEFAULT_CLIENT_SECRET=OTJgUOQcT7lO7GsGZq2G4IlT",
				"HOME=/data/data/org.chromium.Chromium",
				"PULSE_COOKIE=/.hakurei/pulse-cookie",
				"PULSE_SERVER=unix:/run/user/1971/pulse/native",
				"SHELL=/run/current-system/sw/bin/zsh",
				"TERM=xterm-256color",
				"USER=chronos",
				"WAYLAND_DISPLAY=wayland-0",
				"XDG_RUNTIME_DIR=/run/user/1971",
				"XDG_SESSION_CLASS=user",
				"XDG_SESSION_TYPE=wayland",
			},

			// spParamsOp
			Hostname:      "localhost",
			RetainSession: true,
			HostNet:       true,
			HostAbstract:  true,
			Path:          m("/run/current-system/sw/bin/chromium"),
			Args: []string{
				"chromium",
				"--ignore-gpu-blocklist",
				"--disable-smooth-scrolling",
				"--enable-features=UseOzonePlatform",
				"--ozone-platform=wayland",
			},
			SeccompFlags: seccomp.AllowMultiarch,
			Uid:          1971,
			Gid:          100,

			Ops: new(container.Ops).
				// resolveRoot
				Root(m("/var/lib/hakurei/base/org.debian"), std.BindWritable).
				// spParamsOp
				Proc(fhs.AbsProc).
				Tmpfs(hst.AbsPrivateTmp, 1<<12, 0755).
				Bind(fhs.AbsDev, fhs.AbsDev, std.BindWritable|std.BindDevice).
				Tmpfs(fhs.AbsDevShm, 0, 01777).

				// spRuntimeOp
				Tmpfs(fhs.AbsRunUser, 1<<12, 0755).
				Bind(m("/tmp/hakurei.0/runtime/9"), m("/run/user/1971"), std.BindWritable).

				// spTmpdirOp
				Bind(m("/tmp/hakurei.0/tmpdir/9"), fhs.AbsTmp, std.BindWritable).

				// spAccountOp
				Place(m("/etc/passwd"), []byte("chronos:x:1971:100:Hakurei:/data/data/org.chromium.Chromium:/run/current-system/sw/bin/zsh\n")).
				Place(m("/etc/group"), []byte("hakurei:x:100:\n")).

				// spWaylandOp
				Bind(m("/tmp/hakurei.0/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/wayland"), m("/run/user/1971/wayland-0"), 0).

				// spPulseOp
				Bind(m("/run/user/1971/hakurei/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/pulse"), m("/run/user/1971/pulse/native"), 0).
				Place(m("/.hakurei/pulse-cookie"), bytes.Repeat([]byte{0}, pulseCookieSizeMax)).

				// spDBusOp
				Bind(m("/tmp/hakurei.0/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/bus"), m("/run/user/1971/bus"), 0).
				Bind(m("/tmp/hakurei.0/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/system_bus_socket"), m("/var/run/dbus/system_bus_socket"), 0).

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
		}},

		{"nixos permissive defaults no enablements", new(stubNixOS), &hst.Config{Container: &hst.ContainerConfig{
			Filesystem: []hst.FilesystemConfigJSON{
				{FilesystemConfig: &hst.FSBind{
					Target:  fhs.AbsRoot,
					Source:  fhs.AbsRoot,
					Write:   true,
					Special: true,
				}},
				{FilesystemConfig: &hst.FSBind{
					Source:   fhs.AbsDev.Append("kvm"),
					Device:   true,
					Optional: true,
				}},
				{FilesystemConfig: &hst.FSBind{
					Target:  fhs.AbsEtc,
					Source:  fhs.AbsEtc,
					Special: true,
				}},
			},

			Username: "chronos",
			Shell:    m("/run/current-system/sw/bin/zsh"),
			Home:     m("/home/chronos"),

			Path: m("/run/current-system/sw/bin/zsh"),
			Args: []string{"/run/current-system/sw/bin/zsh"},

			Flags: hst.FUserns | hst.FHostNet | hst.FHostAbstract | hst.FTty | hst.FShareRuntime | hst.FShareTmpdir,
		}}, hst.ID{
			0x4a, 0x45, 0x0b, 0x65,
			0x96, 0xd7, 0xbc, 0x15,
			0xbd, 0x01, 0x78, 0x0e,
			0xb9, 0xa6, 0x07, 0xac,
		}, system.New(t.Context(), msg, 10000).
			Ensure(m("/tmp/hakurei.0"), 0711).
			Ensure(m("/tmp/hakurei.0/runtime"), 0700).
			UpdatePermType(system.User, m("/tmp/hakurei.0/runtime"), acl.Execute).
			Ensure(m("/tmp/hakurei.0/runtime/0"), 0700).
			UpdatePermType(system.User, m("/tmp/hakurei.0/runtime/0"), acl.Read, acl.Write, acl.Execute).
			Ensure(m("/tmp/hakurei.0/tmpdir"), 0700).
			UpdatePermType(system.User, m("/tmp/hakurei.0/tmpdir"), acl.Execute).
			Ensure(m("/tmp/hakurei.0/tmpdir/0"), 01700).
			UpdatePermType(system.User, m("/tmp/hakurei.0/tmpdir/0"), acl.Read, acl.Write, acl.Execute), &container.Params{

			Dir:  m("/home/chronos"),
			Path: m("/run/current-system/sw/bin/zsh"),
			Args: []string{"/run/current-system/sw/bin/zsh"},
			Env: []string{
				"HOME=/home/chronos",
				"SHELL=/run/current-system/sw/bin/zsh",
				"TERM=xterm-256color",
				"USER=chronos",
				"XDG_RUNTIME_DIR=/run/user/65534",
				"XDG_SESSION_CLASS=user",
				"XDG_SESSION_TYPE=tty",
			},
			Ops: new(container.Ops).
				Root(m("/"), std.BindWritable).
				Proc(m("/proc/")).
				Tmpfs(hst.AbsPrivateTmp, 4096, 0755).
				DevWritable(m("/dev/"), true).
				Tmpfs(m("/dev/shm/"), 0, 01777).
				Tmpfs(m("/run/user/"), 4096, 0755).
				Bind(m("/tmp/hakurei.0/runtime/0"), m("/run/user/65534"), std.BindWritable).
				Bind(m("/tmp/hakurei.0/tmpdir/0"), m("/tmp/"), std.BindWritable).
				Place(m("/etc/passwd"), []byte("chronos:x:65534:65534:Hakurei:/home/chronos:/run/current-system/sw/bin/zsh\n")).
				Place(m("/etc/group"), []byte("hakurei:x:65534:\n")).
				Bind(m("/dev/kvm"), m("/dev/kvm"), std.BindWritable|std.BindDevice|std.BindOptional).
				Etc(m("/etc/"), "4a450b6596d7bc15bd01780eb9a607ac").
				Tmpfs(m("/run/user/1971"), 8192, 0755).
				Tmpfs(m("/run/nscd"), 8192, 0755).
				Tmpfs(m("/run/dbus"), 8192, 0755).
				Remount(m("/dev/"), syscall.MS_RDONLY).
				Remount(m("/"), syscall.MS_RDONLY),
			SeccompPresets: std.PresetExt | std.PresetDenyDevel,
			HostNet:        true,
			HostAbstract:   true,
			RetainSession:  true,
			ForwardCancel:  true,
		}},

		{"nixos permissive defaults chromium", new(stubNixOS), &hst.Config{
			ID:       "org.chromium.Chromium",
			Identity: 9,
			Groups:   []string{"video"},
			SessionBus: &hst.BusConfig{
				Talk: []string{
					"org.freedesktop.Notifications",
					"org.freedesktop.FileManager1",
					"org.freedesktop.ScreenSaver",
					"org.freedesktop.secrets",
					"org.kde.kwalletd5",
					"org.kde.kwalletd6",
					"org.gnome.SessionManager",
				},
				Own: []string{
					"org.chromium.Chromium.*",
					"org.mpris.MediaPlayer2.org.chromium.Chromium.*",
					"org.mpris.MediaPlayer2.chromium.*",
				},
				Call: map[string]string{
					"org.freedesktop.portal.*": "*",
				},
				Broadcast: map[string]string{
					"org.freedesktop.portal.*": "@/org/freedesktop/portal/*",
				},
				Filter: true,
			},
			SystemBus: &hst.BusConfig{
				Talk: []string{
					"org.bluez",
					"org.freedesktop.Avahi",
					"org.freedesktop.UPower",
				},
				Filter: true,
			},
			Enablements: hst.NewEnablements(hst.EWayland | hst.EDBus | hst.EPulse),

			Container: &hst.ContainerConfig{
				Filesystem: []hst.FilesystemConfigJSON{
					{FilesystemConfig: &hst.FSBind{
						Target:  fhs.AbsRoot,
						Source:  fhs.AbsRoot,
						Write:   true,
						Special: true,
					}},
					{FilesystemConfig: &hst.FSBind{
						Source:   fhs.AbsDev.Append("dri"),
						Device:   true,
						Optional: true,
					}},
					{FilesystemConfig: &hst.FSBind{
						Source:   fhs.AbsDev.Append("kvm"),
						Device:   true,
						Optional: true,
					}},
					{FilesystemConfig: &hst.FSBind{
						Target:  fhs.AbsEtc,
						Source:  fhs.AbsEtc,
						Special: true,
					}},
				},

				Username: "chronos",
				Shell:    m("/run/current-system/sw/bin/zsh"),
				Home:     m("/home/chronos"),

				Path: m("/run/current-system/sw/bin/zsh"),
				Args: []string{"zsh", "-c", "exec chromium "},

				Flags: hst.FUserns | hst.FHostNet | hst.FHostAbstract | hst.FTty | hst.FShareRuntime | hst.FShareTmpdir,
			},
		}, hst.ID{
			0xeb, 0xf0, 0x83, 0xd1,
			0xb1, 0x75, 0x91, 0x17,
			0x82, 0xd4, 0x13, 0x36,
			0x9b, 0x64, 0xce, 0x7c,
		}, system.New(t.Context(), msg, 10009).
			Ensure(m("/tmp/hakurei.0"), 0711).
			Ensure(m("/tmp/hakurei.0/runtime"), 0700).UpdatePermType(system.User, m("/tmp/hakurei.0/runtime"), acl.Execute).
			Ensure(m("/tmp/hakurei.0/runtime/9"), 0700).UpdatePermType(system.User, m("/tmp/hakurei.0/runtime/9"), acl.Read, acl.Write, acl.Execute).
			Ensure(m("/tmp/hakurei.0/tmpdir"), 0700).UpdatePermType(system.User, m("/tmp/hakurei.0/tmpdir"), acl.Execute).
			Ensure(m("/tmp/hakurei.0/tmpdir/9"), 01700).UpdatePermType(system.User, m("/tmp/hakurei.0/tmpdir/9"), acl.Read, acl.Write, acl.Execute).
			Ephemeral(system.Process, m("/tmp/hakurei.0/ebf083d1b175911782d413369b64ce7c"), 0711).
			Wayland(m("/tmp/hakurei.0/ebf083d1b175911782d413369b64ce7c/wayland"), m("/run/user/1971/wayland-0"), "org.chromium.Chromium", "ebf083d1b175911782d413369b64ce7c").
			Ensure(m("/run/user/1971"), 0700).UpdatePermType(system.User, m("/run/user/1971"), acl.Execute). // this is ordered as is because the previous Ensure only calls mkdir if XDG_RUNTIME_DIR is unset
			Ensure(m("/run/user/1971/hakurei"), 0700).UpdatePermType(system.User, m("/run/user/1971/hakurei"), acl.Execute).
			Ephemeral(system.Process, m("/run/user/1971/hakurei/ebf083d1b175911782d413369b64ce7c"), 0700).UpdatePermType(system.Process, m("/run/user/1971/hakurei/ebf083d1b175911782d413369b64ce7c"), acl.Execute).
			Link(m("/run/user/1971/pulse/native"), m("/run/user/1971/hakurei/ebf083d1b175911782d413369b64ce7c/pulse")).
			MustProxyDBus(&hst.BusConfig{
				Talk: []string{
					"org.freedesktop.Notifications",
					"org.freedesktop.FileManager1",
					"org.freedesktop.ScreenSaver",
					"org.freedesktop.secrets",
					"org.kde.kwalletd5",
					"org.kde.kwalletd6",
					"org.gnome.SessionManager",
				},
				Own: []string{
					"org.chromium.Chromium.*",
					"org.mpris.MediaPlayer2.org.chromium.Chromium.*",
					"org.mpris.MediaPlayer2.chromium.*",
				},
				Call: map[string]string{
					"org.freedesktop.portal.*": "*",
				},
				Broadcast: map[string]string{
					"org.freedesktop.portal.*": "@/org/freedesktop/portal/*",
				},
				Filter: true,
			}, &hst.BusConfig{
				Talk: []string{
					"org.bluez",
					"org.freedesktop.Avahi",
					"org.freedesktop.UPower",
				},
				Filter: true,
			}, dbus.ProxyPair{
				"unix:path=/run/user/1971/bus",
				"/tmp/hakurei.0/ebf083d1b175911782d413369b64ce7c/bus",
			}, dbus.ProxyPair{
				"unix:path=/var/run/dbus/system_bus_socket",
				"/tmp/hakurei.0/ebf083d1b175911782d413369b64ce7c/system_bus_socket",
			}).
			UpdatePerm(m("/tmp/hakurei.0/ebf083d1b175911782d413369b64ce7c/bus"), acl.Read, acl.Write).
			UpdatePerm(m("/tmp/hakurei.0/ebf083d1b175911782d413369b64ce7c/system_bus_socket"), acl.Read, acl.Write), &container.Params{

			Dir:  m("/home/chronos"),
			Path: m("/run/current-system/sw/bin/zsh"),
			Args: []string{"zsh", "-c", "exec chromium "},
			Env: []string{
				"DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/65534/bus",
				"DBUS_SYSTEM_BUS_ADDRESS=unix:path=/var/run/dbus/system_bus_socket",
				"HOME=/home/chronos",
				"PULSE_COOKIE=" + hst.PrivateTmp + "/pulse-cookie",
				"PULSE_SERVER=unix:/run/user/65534/pulse/native",
				"SHELL=/run/current-system/sw/bin/zsh",
				"TERM=xterm-256color",
				"USER=chronos",
				"WAYLAND_DISPLAY=wayland-0",
				"XDG_RUNTIME_DIR=/run/user/65534",
				"XDG_SESSION_CLASS=user",
				"XDG_SESSION_TYPE=wayland",
			},
			Ops: new(container.Ops).
				Root(m("/"), std.BindWritable).
				Proc(m("/proc/")).
				Tmpfs(hst.AbsPrivateTmp, 4096, 0755).
				DevWritable(m("/dev/"), true).
				Tmpfs(m("/dev/shm/"), 0, 01777).
				Tmpfs(m("/run/user/"), 4096, 0755).
				Bind(m("/tmp/hakurei.0/runtime/9"), m("/run/user/65534"), std.BindWritable).
				Bind(m("/tmp/hakurei.0/tmpdir/9"), m("/tmp/"), std.BindWritable).
				Place(m("/etc/passwd"), []byte("chronos:x:65534:65534:Hakurei:/home/chronos:/run/current-system/sw/bin/zsh\n")).
				Place(m("/etc/group"), []byte("hakurei:x:65534:\n")).
				Bind(m("/tmp/hakurei.0/ebf083d1b175911782d413369b64ce7c/wayland"), m("/run/user/65534/wayland-0"), 0).
				Bind(m("/run/user/1971/hakurei/ebf083d1b175911782d413369b64ce7c/pulse"), m("/run/user/65534/pulse/native"), 0).
				Place(m(hst.PrivateTmp+"/pulse-cookie"), bytes.Repeat([]byte{0}, pulseCookieSizeMax)).
				Bind(m("/tmp/hakurei.0/ebf083d1b175911782d413369b64ce7c/bus"), m("/run/user/65534/bus"), 0).
				Bind(m("/tmp/hakurei.0/ebf083d1b175911782d413369b64ce7c/system_bus_socket"), m("/var/run/dbus/system_bus_socket"), 0).
				Bind(m("/dev/dri"), m("/dev/dri"), std.BindWritable|std.BindDevice|std.BindOptional).
				Bind(m("/dev/kvm"), m("/dev/kvm"), std.BindWritable|std.BindDevice|std.BindOptional).
				Etc(m("/etc/"), "ebf083d1b175911782d413369b64ce7c").
				Tmpfs(m("/run/user/1971"), 8192, 0755).
				Tmpfs(m("/run/nscd"), 8192, 0755).
				Tmpfs(m("/run/dbus"), 8192, 0755).
				Remount(m("/dev/"), syscall.MS_RDONLY).
				Remount(m("/"), syscall.MS_RDONLY),
			SeccompPresets: std.PresetExt | std.PresetDenyDevel,
			HostNet:        true,
			HostAbstract:   true,
			RetainSession:  true,
			ForwardCancel:  true,
		}},

		{"nixos chromium direct wayland", new(stubNixOS), &hst.Config{
			ID:          "org.chromium.Chromium",
			Enablements: hst.NewEnablements(hst.EWayland | hst.EDBus | hst.EPulse),
			Container: &hst.ContainerConfig{
				Env: nil,
				Filesystem: []hst.FilesystemConfigJSON{
					f(&hst.FSBind{Source: m("/bin")}),
					f(&hst.FSBind{Source: m("/usr/bin/")}),
					f(&hst.FSBind{Source: m("/nix/store")}),
					f(&hst.FSBind{Source: m("/run/current-system")}),
					f(&hst.FSBind{Source: m("/sys/block"), Optional: true}),
					f(&hst.FSBind{Source: m("/sys/bus"), Optional: true}),
					f(&hst.FSBind{Source: m("/sys/class"), Optional: true}),
					f(&hst.FSBind{Source: m("/sys/dev"), Optional: true}),
					f(&hst.FSBind{Source: m("/sys/devices"), Optional: true}),
					f(&hst.FSBind{Source: m("/run/opengl-driver")}),
					f(&hst.FSBind{Source: m("/dev/dri"), Device: true, Optional: true}),
					f(&hst.FSBind{Source: m("/etc/"), Target: m("/etc/"), Special: true}),
					f(&hst.FSBind{Source: m("/var/lib/persist/module/hakurei/0/1"), Write: true, Ensure: true}),
				},

				Username: "u0_a1",
				Shell:    m("/run/current-system/sw/bin/zsh"),
				Home:     m("/var/lib/persist/module/hakurei/0/1"),

				Path: m("/nix/store/yqivzpzzn7z5x0lq9hmbzygh45d8rhqd-chromium-start"),

				Flags: hst.FUserns | hst.FHostNet | hst.FMapRealUID | hst.FShareRuntime | hst.FShareTmpdir,
			},
			SystemBus: &hst.BusConfig{
				Talk:   []string{"org.bluez", "org.freedesktop.Avahi", "org.freedesktop.UPower"},
				Filter: true,
			},
			SessionBus: &hst.BusConfig{
				Talk: []string{
					"org.freedesktop.FileManager1", "org.freedesktop.Notifications",
					"org.freedesktop.ScreenSaver", "org.freedesktop.secrets",
					"org.kde.kwalletd5", "org.kde.kwalletd6",
				},
				Own: []string{
					"org.chromium.Chromium.*",
					"org.mpris.MediaPlayer2.org.chromium.Chromium.*",
					"org.mpris.MediaPlayer2.chromium.*",
				},
				Call: map[string]string{}, Broadcast: map[string]string{},
				Filter: true,
			},
			DirectWayland: true,

			Identity: 1, Groups: []string{},
		}, hst.ID{
			0x8e, 0x2c, 0x76, 0xb0,
			0x66, 0xda, 0xbe, 0x57,
			0x4c, 0xf0, 0x73, 0xbd,
			0xb4, 0x6e, 0xb5, 0xc1,
		}, system.New(t.Context(), msg, 10001).
			Ensure(m("/tmp/hakurei.0"), 0711).
			Ensure(m("/tmp/hakurei.0/runtime"), 0700).UpdatePermType(system.User, m("/tmp/hakurei.0/runtime"), acl.Execute).
			Ensure(m("/tmp/hakurei.0/runtime/1"), 0700).UpdatePermType(system.User, m("/tmp/hakurei.0/runtime/1"), acl.Read, acl.Write, acl.Execute).
			Ensure(m("/tmp/hakurei.0/tmpdir"), 0700).UpdatePermType(system.User, m("/tmp/hakurei.0/tmpdir"), acl.Execute).
			Ensure(m("/tmp/hakurei.0/tmpdir/1"), 01700).UpdatePermType(system.User, m("/tmp/hakurei.0/tmpdir/1"), acl.Read, acl.Write, acl.Execute).
			Ensure(m("/run/user/1971"), 0700).UpdatePermType(system.User, m("/run/user/1971"), acl.Execute). // this is ordered as is because the previous Ensure only calls mkdir if XDG_RUNTIME_DIR is unset
			Ensure(m("/run/user/1971/hakurei"), 0700).UpdatePermType(system.User, m("/run/user/1971/hakurei"), acl.Execute).
			UpdatePermType(hst.EWayland, m("/run/user/1971/wayland-0"), acl.Read, acl.Write, acl.Execute).
			Ephemeral(system.Process, m("/run/user/1971/hakurei/8e2c76b066dabe574cf073bdb46eb5c1"), 0700).UpdatePermType(system.Process, m("/run/user/1971/hakurei/8e2c76b066dabe574cf073bdb46eb5c1"), acl.Execute).
			Link(m("/run/user/1971/pulse/native"), m("/run/user/1971/hakurei/8e2c76b066dabe574cf073bdb46eb5c1/pulse")).
			Ephemeral(system.Process, m("/tmp/hakurei.0/8e2c76b066dabe574cf073bdb46eb5c1"), 0711).
			MustProxyDBus(&hst.BusConfig{
				Talk: []string{
					"org.freedesktop.FileManager1", "org.freedesktop.Notifications",
					"org.freedesktop.ScreenSaver", "org.freedesktop.secrets",
					"org.kde.kwalletd5", "org.kde.kwalletd6",
				},
				Own: []string{
					"org.chromium.Chromium.*",
					"org.mpris.MediaPlayer2.org.chromium.Chromium.*",
					"org.mpris.MediaPlayer2.chromium.*",
				},
				Call: map[string]string{}, Broadcast: map[string]string{},
				Filter: true,
			}, &hst.BusConfig{
				Talk: []string{
					"org.bluez",
					"org.freedesktop.Avahi",
					"org.freedesktop.UPower",
				},
				Filter: true,
			}, dbus.ProxyPair{
				"unix:path=/run/user/1971/bus",
				"/tmp/hakurei.0/8e2c76b066dabe574cf073bdb46eb5c1/bus",
			}, dbus.ProxyPair{
				"unix:path=/var/run/dbus/system_bus_socket",
				"/tmp/hakurei.0/8e2c76b066dabe574cf073bdb46eb5c1/system_bus_socket",
			}).
			UpdatePerm(m("/tmp/hakurei.0/8e2c76b066dabe574cf073bdb46eb5c1/bus"), acl.Read, acl.Write).
			UpdatePerm(m("/tmp/hakurei.0/8e2c76b066dabe574cf073bdb46eb5c1/system_bus_socket"), acl.Read, acl.Write), &container.Params{

			Uid:  1971,
			Gid:  100,
			Dir:  m("/var/lib/persist/module/hakurei/0/1"),
			Path: m("/nix/store/yqivzpzzn7z5x0lq9hmbzygh45d8rhqd-chromium-start"),
			Args: []string{"/nix/store/yqivzpzzn7z5x0lq9hmbzygh45d8rhqd-chromium-start"},
			Env: []string{
				"DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1971/bus",
				"DBUS_SYSTEM_BUS_ADDRESS=unix:path=/var/run/dbus/system_bus_socket",
				"HOME=/var/lib/persist/module/hakurei/0/1",
				"PULSE_COOKIE=" + hst.PrivateTmp + "/pulse-cookie",
				"PULSE_SERVER=unix:/run/user/1971/pulse/native",
				"SHELL=/run/current-system/sw/bin/zsh",
				"TERM=xterm-256color",
				"USER=u0_a1",
				"WAYLAND_DISPLAY=wayland-0",
				"XDG_RUNTIME_DIR=/run/user/1971",
				"XDG_SESSION_CLASS=user",
				"XDG_SESSION_TYPE=wayland",
			},
			Ops: new(container.Ops).
				Proc(m("/proc/")).
				Tmpfs(hst.AbsPrivateTmp, 4096, 0755).
				DevWritable(m("/dev/"), true).
				Tmpfs(m("/dev/shm/"), 0, 01777).
				Tmpfs(m("/run/user/"), 4096, 0755).
				Bind(m("/tmp/hakurei.0/runtime/1"), m("/run/user/1971"), std.BindWritable).
				Bind(m("/tmp/hakurei.0/tmpdir/1"), m("/tmp/"), std.BindWritable).
				Place(m("/etc/passwd"), []byte("u0_a1:x:1971:100:Hakurei:/var/lib/persist/module/hakurei/0/1:/run/current-system/sw/bin/zsh\n")).
				Place(m("/etc/group"), []byte("hakurei:x:100:\n")).
				Bind(m("/run/user/1971/wayland-0"), m("/run/user/1971/wayland-0"), 0).
				Bind(m("/run/user/1971/hakurei/8e2c76b066dabe574cf073bdb46eb5c1/pulse"), m("/run/user/1971/pulse/native"), 0).
				Place(m(hst.PrivateTmp+"/pulse-cookie"), bytes.Repeat([]byte{0}, pulseCookieSizeMax)).
				Bind(m("/tmp/hakurei.0/8e2c76b066dabe574cf073bdb46eb5c1/bus"), m("/run/user/1971/bus"), 0).
				Bind(m("/tmp/hakurei.0/8e2c76b066dabe574cf073bdb46eb5c1/system_bus_socket"), m("/var/run/dbus/system_bus_socket"), 0).
				Bind(m("/bin"), m("/bin"), 0).
				Bind(m("/usr/bin/"), m("/usr/bin/"), 0).
				Bind(m("/nix/store"), m("/nix/store"), 0).
				Bind(m("/run/current-system"), m("/run/current-system"), 0).
				Bind(m("/sys/block"), m("/sys/block"), std.BindOptional).
				Bind(m("/sys/bus"), m("/sys/bus"), std.BindOptional).
				Bind(m("/sys/class"), m("/sys/class"), std.BindOptional).
				Bind(m("/sys/dev"), m("/sys/dev"), std.BindOptional).
				Bind(m("/sys/devices"), m("/sys/devices"), std.BindOptional).
				Bind(m("/run/opengl-driver"), m("/run/opengl-driver"), 0).
				Bind(m("/dev/dri"), m("/dev/dri"), std.BindDevice|std.BindWritable|std.BindOptional).
				Etc(m("/etc/"), "8e2c76b066dabe574cf073bdb46eb5c1").
				Bind(m("/var/lib/persist/module/hakurei/0/1"), m("/var/lib/persist/module/hakurei/0/1"), std.BindWritable|std.BindEnsure).
				Remount(m("/dev/"), syscall.MS_RDONLY).
				Remount(m("/"), syscall.MS_RDONLY),
			SeccompPresets: std.PresetExt | std.PresetDenyTTY | std.PresetDenyDevel,
			HostNet:        true,
			ForwardCancel:  true,
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gr, gw := io.Pipe()

			var gotSys *system.I
			{
				sPriv := newOutcomeState(tc.k, msg, &tc.id, tc.config, &Hsu{k: tc.k})
				if err := sPriv.populateLocal(tc.k, msg); err != nil {
					t.Fatalf("populateLocal: error = %#v", err)
				}

				gotSys = system.New(t.Context(), msg, sPriv.uid.unwrap())
				if err := sPriv.newSys(tc.config, gotSys).toSystem(); err != nil {
					t.Fatalf("toSystem: error = %#v", err)
				}

				go func() {
					e := gob.NewEncoder(gw)
					if err := errors.Join(e.Encode(&sPriv)); err != nil {
						t.Errorf("Encode: error = %v", err)
						panic("unexpected encode fault")
					}
				}()
			}

			var gotParams *container.Params
			{
				var sShim outcomeState

				d := gob.NewDecoder(gr)
				if err := errors.Join(d.Decode(&sShim)); err != nil {
					t.Fatalf("Decode: error = %v", err)
				}
				if err := sShim.populateLocal(tc.k, msg); err != nil {
					t.Fatalf("populateLocal: error = %#v", err)
				}

				stateParams := sShim.newParams()
				for _, op := range sShim.Shim.Ops {
					if err := op.toContainer(stateParams); err != nil {
						t.Fatalf("toContainer: error = %#v", err)
					}
				}
				gotParams = stateParams.params
			}

			t.Run("sys", func(t *testing.T) {
				if !gotSys.Equal(tc.wantSys) {
					t.Errorf("toSystem: sys = %#v, want %#v", gotSys, tc.wantSys)
				}
			})

			t.Run("params", func(t *testing.T) {
				if !reflect.DeepEqual(gotParams, tc.wantParams) {
					t.Errorf("toContainer: params =\n%s\n, want\n%s", mustMarshal(gotParams), mustMarshal(tc.wantParams))
				}
			})
		})
	}
}

func stubDirEntries(names ...string) (e []fs.DirEntry, err error) {
	e = make([]fs.DirEntry, len(names))
	for i, name := range names {
		e[i] = stubDirEntryPath(name)
	}
	return
}

type stubDirEntryPath string

func (p stubDirEntryPath) Name() string               { return string(p) }
func (p stubDirEntryPath) IsDir() bool                { panic("attempted to call IsDir") }
func (p stubDirEntryPath) Type() fs.FileMode          { panic("attempted to call Type") }
func (p stubDirEntryPath) Info() (fs.FileInfo, error) { panic("attempted to call Info") }

type stubFileInfoMode fs.FileMode

func (s stubFileInfoMode) Name() string       { panic("attempted to call Name") }
func (s stubFileInfoMode) Size() int64        { panic("attempted to call Size") }
func (s stubFileInfoMode) Mode() fs.FileMode  { return fs.FileMode(s) }
func (s stubFileInfoMode) ModTime() time.Time { panic("attempted to call ModTime") }
func (s stubFileInfoMode) IsDir() bool        { panic("attempted to call IsDir") }
func (s stubFileInfoMode) Sys() any           { panic("attempted to call Sys") }

type stubFileInfoIsDir bool

func (s stubFileInfoIsDir) Name() string       { panic("attempted to call Name") }
func (s stubFileInfoIsDir) Size() int64        { panic("attempted to call Size") }
func (s stubFileInfoIsDir) Mode() fs.FileMode  { panic("attempted to call Mode") }
func (s stubFileInfoIsDir) ModTime() time.Time { panic("attempted to call ModTime") }
func (s stubFileInfoIsDir) IsDir() bool        { return bool(s) }
func (s stubFileInfoIsDir) Sys() any           { panic("attempted to call Sys") }

type stubFileInfoPulseCookie struct{ stubFileInfoIsDir }

func (s stubFileInfoPulseCookie) Size() int64 { return pulseCookieSizeMax }

type stubOsFileReadCloser struct{ io.ReadCloser }

func (s stubOsFileReadCloser) Name() string               { panic("attempting to call Name") }
func (s stubOsFileReadCloser) Write([]byte) (int, error)  { panic("attempting to call Write") }
func (s stubOsFileReadCloser) Stat() (fs.FileInfo, error) { panic("attempting to call Stat") }

type stubNixOS struct {
	usernameErr map[string]error
	panicDispatcher
}

func (k *stubNixOS) getppid() int { return 0xbad }
func (k *stubNixOS) getpid() int  { return 0xdead }
func (k *stubNixOS) getuid() int  { return 1971 }
func (k *stubNixOS) getgid() int  { return 100 }

func (k *stubNixOS) lookupEnv(key string) (string, bool) {
	switch key {
	case "SHELL":
		return "/run/current-system/sw/bin/zsh", true
	case "TERM":
		return "xterm-256color", true
	case "WAYLAND_DISPLAY":
		return "wayland-0", true
	case "PULSE_COOKIE":
		return "", false
	case "HOME":
		return "/home/ophestra", true
	case "XDG_RUNTIME_DIR":
		return "/run/user/1971", true
	case "XDG_CONFIG_HOME":
		return "/home/ophestra/xdg/config", true
	case "DBUS_SYSTEM_BUS_ADDRESS":
		return "", false
	default:
		panic(fmt.Sprintf("attempted to access unexpected environment variable %q", key))
	}
}

func (k *stubNixOS) stat(name string) (fs.FileInfo, error) {
	switch name {
	case "/var/run/nscd":
		return nil, nil
	case "/run/user/1971/pulse":
		return nil, nil
	case "/run/user/1971/pulse/native":
		return stubFileInfoMode(0666), nil
	case "/home/ophestra/.pulse-cookie":
		return stubFileInfoIsDir(true), nil
	case "/home/ophestra/xdg/config/pulse/cookie":
		return stubFileInfoPulseCookie{false}, nil
	default:
		panic(fmt.Sprintf("attempted to stat unexpected path %q", name))
	}
}

func (k *stubNixOS) open(name string) (osFile, error) {
	switch name {
	case "/home/ophestra/xdg/config/pulse/cookie":
		return stubOsFileReadCloser{io.NopCloser(bytes.NewReader(bytes.Repeat([]byte{0}, pulseCookieSizeMax)))}, nil
	default:
		panic(fmt.Sprintf("attempted to open unexpected path %q", name))
	}
}

func (k *stubNixOS) readdir(name string) ([]fs.DirEntry, error) {
	switch name {
	case "/":
		return stubDirEntries("bin", "boot", "dev", "etc", "home", "lib",
			"lib64", "nix", "proc", "root", "run", "srv", "sys", "tmp", "usr", "var")

	case "/run":
		return stubDirEntries("agetty.reload", "binfmt", "booted-system",
			"credentials", "cryptsetup", "current-system", "dbus", "host", "keys",
			"libvirt", "libvirtd.pid", "lock", "log", "lvm", "mount", "NetworkManager",
			"nginx", "nixos", "nscd", "opengl-driver", "pppd", "resolvconf", "sddm",
			"store", "syncoid", "system", "systemd", "tmpfiles.d", "udev", "udisks2",
			"user", "utmp", "virtlogd.pid", "wrappers", "zed.pid", "zed.state")

	case "/etc":
		return stubDirEntries("alsa", "bashrc", "binfmt.d", "dbus-1", "default",
			"ethertypes", "fonts", "fstab", "fuse.conf", "group", "host.conf", "hostid",
			"hostname", "hostname.CHECKSUM", "hosts", "inputrc", "ipsec.d", "issue", "kbd",
			"libblockdev", "locale.conf", "localtime", "login.defs", "lsb-release", "lvm",
			"machine-id", "man_db.conf", "modprobe.d", "modules-load.d", "mtab", "nanorc",
			"netgroup", "NetworkManager", "nix", "nixos", "NIXOS", "nscd.conf", "nsswitch.conf",
			"opensnitchd", "os-release", "pam", "pam.d", "passwd", "pipewire", "pki", "polkit-1",
			"profile", "protocols", "qemu", "resolv.conf", "resolvconf.conf", "rpc", "samba",
			"sddm.conf", "secureboot", "services", "set-environment", "shadow", "shells", "ssh",
			"ssl", "static", "subgid", "subuid", "sudoers", "sysctl.d", "systemd", "terminfo",
			"tmpfiles.d", "udev", "udisks2", "UPower", "vconsole.conf", "X11", "zfs", "zinputrc",
			"zoneinfo", "zprofile", "zshenv", "zshrc")

	case "/var/lib/hakurei/base/org.debian":
		return stubDirEntries("bin", "dev", "etc", "home", "lib64", "lost+found",
			"mnt", "nix", "proc", "root", "run", "srv", "sys", "tmp", "usr", "var")

	default:
		panic(fmt.Sprintf("attempted to read unexpected directory %q", name))
	}
}

func (k *stubNixOS) tempdir() string { return "/tmp/" }

func (k *stubNixOS) evalSymlinks(path string) (string, error) {
	switch path {
	case "/var/run/nscd":
		return "/run/nscd", nil
	case "/run/user/1971":
		return "/run/user/1971", nil
	case "/tmp/hakurei.0":
		return "/tmp/hakurei.0", nil
	case "/var/run/dbus":
		return "/run/dbus", nil
	case "/dev/kvm":
		return "/dev/kvm", nil
	case "/etc/":
		return "/etc/", nil
	case "/bin":
		return "/bin", nil
	case "/boot":
		return "/boot", nil
	case "/home":
		return "/home", nil
	case "/lib":
		return "/lib", nil
	case "/lib64":
		return "/lib64", nil
	case "/nix":
		return "/nix", nil
	case "/root":
		return "/root", nil
	case "/run":
		return "/run", nil
	case "/srv":
		return "/srv", nil
	case "/sys":
		return "/sys", nil
	case "/usr":
		return "/usr", nil
	case "/var":
		return "/var", nil
	case "/dev/dri":
		return "/dev/dri", nil
	case "/usr/bin/":
		return "/usr/bin/", nil
	case "/nix/store":
		return "/nix/store", nil
	case "/run/current-system":
		return "/nix/store/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-nixos-system-satori-25.05.99999999.aaaaaaa", nil
	case "/sys/block":
		return "/sys/block", nil
	case "/sys/bus":
		return "/sys/bus", nil
	case "/sys/class":
		return "/sys/class", nil
	case "/sys/dev":
		return "/sys/dev", nil
	case "/sys/devices":
		return "/sys/devices", nil
	case "/run/opengl-driver":
		return "/nix/store/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-graphics-drivers", nil
	case "/var/lib/persist/module/hakurei/0/1":
		return "/var/lib/persist/module/hakurei/0/1", nil

	case "/var/lib/hakurei/nix/u0/org.chromium.Chromium/rw-store/upper":
		return "/var/lib/hakurei/nix/u0/org.chromium.Chromium/rw-store/upper", nil
	case "/var/lib/hakurei/nix/u0/org.chromium.Chromium/rw-store/work":
		return "/var/lib/hakurei/nix/u0/org.chromium.Chromium/rw-store/work", nil
	case "/var/lib/hakurei/base/org.nixos/ro-store":
		return "/var/lib/hakurei/base/org.nixos/ro-store", nil
	case "/var/lib/hakurei/u0/org.chromium.Chromium":
		return "/var/lib/hakurei/u0/org.chromium.Chromium", nil
	case "/var/lib/hakurei/base/org.debian/bin":
		return "/var/lib/hakurei/base/org.debian/bin", nil
	case "/var/lib/hakurei/base/org.debian/home":
		return "/var/lib/hakurei/base/org.debian/home", nil
	case "/var/lib/hakurei/base/org.debian/lib64":
		return "/var/lib/hakurei/base/org.debian/lib64", nil
	case "/var/lib/hakurei/base/org.debian/lost+found":
		return "/var/lib/hakurei/base/org.debian/lost+found", nil
	case "/var/lib/hakurei/base/org.debian/nix":
		return "/var/lib/hakurei/base/org.debian/nix", nil
	case "/var/lib/hakurei/base/org.debian/root":
		return "/var/lib/hakurei/base/org.debian/root", nil
	case "/var/lib/hakurei/base/org.debian/run":
		return "/var/lib/hakurei/base/org.debian/run", nil
	case "/var/lib/hakurei/base/org.debian/srv":
		return "/var/lib/hakurei/base/org.debian/srv", nil
	case "/var/lib/hakurei/base/org.debian/sys":
		return "/var/lib/hakurei/base/org.debian/sys", nil
	case "/var/lib/hakurei/base/org.debian/usr":
		return "/var/lib/hakurei/base/org.debian/usr", nil
	case "/var/lib/hakurei/base/org.debian/var":
		return "/var/lib/hakurei/base/org.debian/var", nil

	default:
		panic(fmt.Sprintf("attempted to evaluate unexpected path %q", path))
	}
}

func (k *stubNixOS) lookupGroupId(name string) (string, error) {
	switch name {
	case "video":
		return "26", nil
	default:
		return "", user.UnknownGroupError(name)
	}
}

func (k *stubNixOS) cmdOutput(cmd *exec.Cmd) ([]byte, error) {
	switch cmd.Path {
	case "/proc/nonexistent/hsu":
		return []byte{'0'}, nil
	default:
		panic(fmt.Sprintf("unexpected cmd %#v", cmd))
	}
}

func (k *stubNixOS) overflowUid(message.Msg) int { return 65534 }
func (k *stubNixOS) overflowGid(message.Msg) int { return 65534 }

func (k *stubNixOS) mustHsuPath() *check.Absolute { return m("/proc/nonexistent/hsu") }

func (k *stubNixOS) dbusAddress() (string, string) {
	return "unix:path=/run/user/1971/bus", "unix:path=/var/run/dbus/system_bus_socket"
}

func (k *stubNixOS) fatalf(format string, v ...any) { panic(fmt.Sprintf(format, v...)) }

func (k *stubNixOS) isVerbose() bool                  { return true }
func (k *stubNixOS) verbose(v ...any)                 { log.Print(v...) }
func (k *stubNixOS) verbosef(format string, v ...any) { log.Printf(format, v...) }
