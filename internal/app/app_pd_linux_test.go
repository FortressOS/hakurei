package app_test

import (
	"os"
	"syscall"

	"hakurei.app/container"
	"hakurei.app/container/seccomp"
	"hakurei.app/hst"
	"hakurei.app/internal/app/state"
	"hakurei.app/system"
	"hakurei.app/system/acl"
	"hakurei.app/system/dbus"
)

var testCasesPd = []sealTestCase{
	{
		"nixos permissive defaults no enablements", new(stubNixOS),
		&hst.Config{Username: "chronos", Data: m("/home/chronos")},
		state.ID{
			0x4a, 0x45, 0x0b, 0x65,
			0x96, 0xd7, 0xbc, 0x15,
			0xbd, 0x01, 0x78, 0x0e,
			0xb9, 0xa6, 0x07, 0xac,
		},
		system.New(1000000).
			Ensure("/tmp/hakurei.1971", 0711).
			Ensure("/tmp/hakurei.1971/runtime", 0700).UpdatePermType(system.User, "/tmp/hakurei.1971/runtime", acl.Execute).
			Ensure("/tmp/hakurei.1971/runtime/0", 0700).UpdatePermType(system.User, "/tmp/hakurei.1971/runtime/0", acl.Read, acl.Write, acl.Execute).
			Ensure("/tmp/hakurei.1971/tmpdir", 0700).UpdatePermType(system.User, "/tmp/hakurei.1971/tmpdir", acl.Execute).
			Ensure("/tmp/hakurei.1971/tmpdir/0", 01700).UpdatePermType(system.User, "/tmp/hakurei.1971/tmpdir/0", acl.Read, acl.Write, acl.Execute),
		&container.Params{
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
				Root(m("/"), "4a450b6596d7bc15bd01780eb9a607ac", container.BindWritable).
				Proc(m("/proc/")).
				Tmpfs(hst.AbsTmp, 4096, 0755).
				DevWritable(m("/dev/"), true).
				Bind(m("/dev/kvm"), m("/dev/kvm"), container.BindWritable|container.BindDevice|container.BindOptional).
				Readonly(m("/var/run/nscd"), 0755).
				Tmpfs(m("/run/user/1971"), 8192, 0755).
				Tmpfs(m("/run/dbus"), 8192, 0755).
				Etc(m("/etc/"), "4a450b6596d7bc15bd01780eb9a607ac").
				Remount(m("/dev/"), syscall.MS_RDONLY).
				Tmpfs(m("/run/user/"), 4096, 0755).
				Bind(m("/tmp/hakurei.1971/runtime/0"), m("/run/user/65534"), container.BindWritable).
				Bind(m("/tmp/hakurei.1971/tmpdir/0"), m("/tmp/"), container.BindWritable).
				Bind(m("/home/chronos"), m("/home/chronos"), container.BindWritable).
				Place(m("/etc/passwd"), []byte("chronos:x:65534:65534:Hakurei:/home/chronos:/run/current-system/sw/bin/zsh\n")).
				Place(m("/etc/group"), []byte("hakurei:x:65534:\n")).
				Remount(m("/"), syscall.MS_RDONLY),
			SeccompPresets: seccomp.PresetExt | seccomp.PresetDenyDevel,
			HostNet:        true,
			RetainSession:  true,
			ForwardCancel:  true,
		},
	},
	{
		"nixos permissive defaults chromium", new(stubNixOS),
		&hst.Config{
			ID:       "org.chromium.Chromium",
			Args:     []string{"zsh", "-c", "exec chromium "},
			Identity: 9,
			Groups:   []string{"video"},
			Username: "chronos",
			Data:     m("/home/chronos"),
			SessionBus: &dbus.Config{
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
			SystemBus: &dbus.Config{
				Talk: []string{
					"org.bluez",
					"org.freedesktop.Avahi",
					"org.freedesktop.UPower",
				},
				Filter: true,
			},
			Enablements: system.EWayland | system.EDBus | system.EPulse,
		},
		state.ID{
			0xeb, 0xf0, 0x83, 0xd1,
			0xb1, 0x75, 0x91, 0x17,
			0x82, 0xd4, 0x13, 0x36,
			0x9b, 0x64, 0xce, 0x7c,
		},
		system.New(1000009).
			Ensure("/tmp/hakurei.1971", 0711).
			Ensure("/tmp/hakurei.1971/runtime", 0700).UpdatePermType(system.User, "/tmp/hakurei.1971/runtime", acl.Execute).
			Ensure("/tmp/hakurei.1971/runtime/9", 0700).UpdatePermType(system.User, "/tmp/hakurei.1971/runtime/9", acl.Read, acl.Write, acl.Execute).
			Ensure("/tmp/hakurei.1971/tmpdir", 0700).UpdatePermType(system.User, "/tmp/hakurei.1971/tmpdir", acl.Execute).
			Ensure("/tmp/hakurei.1971/tmpdir/9", 01700).UpdatePermType(system.User, "/tmp/hakurei.1971/tmpdir/9", acl.Read, acl.Write, acl.Execute).
			Ephemeral(system.Process, "/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c", 0711).
			Wayland(new(*os.File), "/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland", "/run/user/1971/wayland-0", "org.chromium.Chromium", "ebf083d1b175911782d413369b64ce7c").
			Ensure("/run/user/1971/hakurei", 0700).UpdatePermType(system.User, "/run/user/1971/hakurei", acl.Execute).
			Ensure("/run/user/1971", 0700).UpdatePermType(system.User, "/run/user/1971", acl.Execute). // this is ordered as is because the previous Ensure only calls mkdir if XDG_RUNTIME_DIR is unset
			Ephemeral(system.Process, "/run/user/1971/hakurei/ebf083d1b175911782d413369b64ce7c", 0700).UpdatePermType(system.Process, "/run/user/1971/hakurei/ebf083d1b175911782d413369b64ce7c", acl.Execute).
			Link("/run/user/1971/pulse/native", "/run/user/1971/hakurei/ebf083d1b175911782d413369b64ce7c/pulse").
			CopyFile(new([]byte), "/home/ophestra/xdg/config/pulse/cookie", 256, 256).
			MustProxyDBus("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/bus", &dbus.Config{
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
			}, "/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/system_bus_socket", &dbus.Config{
				Talk: []string{
					"org.bluez",
					"org.freedesktop.Avahi",
					"org.freedesktop.UPower",
				},
				Filter: true,
			}).
			UpdatePerm("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/bus", acl.Read, acl.Write).
			UpdatePerm("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/system_bus_socket", acl.Read, acl.Write),
		&container.Params{
			Dir:  m("/home/chronos"),
			Path: m("/run/current-system/sw/bin/zsh"),
			Args: []string{"zsh", "-c", "exec chromium "},
			Env: []string{
				"DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/65534/bus",
				"DBUS_SYSTEM_BUS_ADDRESS=unix:path=/run/dbus/system_bus_socket",
				"HOME=/home/chronos",
				"PULSE_COOKIE=" + hst.Tmp + "/pulse-cookie",
				"PULSE_SERVER=unix:/run/user/65534/pulse/native",
				"SHELL=/run/current-system/sw/bin/zsh",
				"TERM=xterm-256color",
				"USER=chronos",
				"WAYLAND_DISPLAY=wayland-0",
				"XDG_RUNTIME_DIR=/run/user/65534",
				"XDG_SESSION_CLASS=user",
				"XDG_SESSION_TYPE=tty",
			},
			Ops: new(container.Ops).
				Root(m("/"), "ebf083d1b175911782d413369b64ce7c", container.BindWritable).
				Proc(m("/proc/")).
				Tmpfs(hst.AbsTmp, 4096, 0755).
				DevWritable(m("/dev/"), true).
				Bind(m("/dev/dri"), m("/dev/dri"), container.BindWritable|container.BindDevice|container.BindOptional).
				Bind(m("/dev/kvm"), m("/dev/kvm"), container.BindWritable|container.BindDevice|container.BindOptional).
				Readonly(m("/var/run/nscd"), 0755).
				Tmpfs(m("/run/user/1971"), 8192, 0755).
				Tmpfs(m("/run/dbus"), 8192, 0755).
				Etc(m("/etc/"), "ebf083d1b175911782d413369b64ce7c").
				Remount(m("/dev/"), syscall.MS_RDONLY).
				Tmpfs(m("/run/user/"), 4096, 0755).
				Bind(m("/tmp/hakurei.1971/runtime/9"), m("/run/user/65534"), container.BindWritable).
				Bind(m("/tmp/hakurei.1971/tmpdir/9"), m("/tmp/"), container.BindWritable).
				Bind(m("/home/chronos"), m("/home/chronos"), container.BindWritable).
				Place(m("/etc/passwd"), []byte("chronos:x:65534:65534:Hakurei:/home/chronos:/run/current-system/sw/bin/zsh\n")).
				Place(m("/etc/group"), []byte("hakurei:x:65534:\n")).
				Bind(m("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"), m("/run/user/65534/wayland-0"), 0).
				Bind(m("/run/user/1971/hakurei/ebf083d1b175911782d413369b64ce7c/pulse"), m("/run/user/65534/pulse/native"), 0).
				Place(m(hst.Tmp+"/pulse-cookie"), nil).
				Bind(m("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/bus"), m("/run/user/65534/bus"), 0).
				Bind(m("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/system_bus_socket"), m("/run/dbus/system_bus_socket"), 0).
				Remount(m("/"), syscall.MS_RDONLY),
			SeccompPresets: seccomp.PresetExt | seccomp.PresetDenyDevel,
			HostNet:        true,
			RetainSession:  true,
			ForwardCancel:  true,
		},
	},
}
