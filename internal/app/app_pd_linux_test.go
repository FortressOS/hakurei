package app_test

import (
	"os"

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
		&hst.Config{Username: "chronos", Data: "/home/chronos"},
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
			Dir:  "/home/chronos",
			Path: "/run/current-system/sw/bin/zsh",
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
				Root("/", "4a450b6596d7bc15bd01780eb9a607ac", container.BindWritable).
				Proc("/proc").
				Tmpfs(hst.Tmp, 4096, 0755).
				Dev("/dev").Mqueue("/dev/mqueue").
				Bind("/dev/kvm", "/dev/kvm", container.BindWritable|container.BindDevice|container.BindOptional).
				Readonly("/var/run/nscd", 0755).
				Tmpfs("/run/user/1971", 8192, 0755).
				Tmpfs("/run/dbus", 8192, 0755).
				Etc("/etc", "4a450b6596d7bc15bd01780eb9a607ac").
				Tmpfs("/run/user", 4096, 0755).
				Bind("/tmp/hakurei.1971/runtime/0", "/run/user/65534", container.BindWritable).
				Bind("/tmp/hakurei.1971/tmpdir/0", "/tmp", container.BindWritable).
				Bind("/home/chronos", "/home/chronos", container.BindWritable).
				Place("/etc/passwd", []byte("chronos:x:65534:65534:Hakurei:/home/chronos:/run/current-system/sw/bin/zsh\n")).
				Place("/etc/group", []byte("hakurei:x:65534:\n")),
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
			Data:     "/home/chronos",
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
			Dir:  "/home/chronos",
			Path: "/run/current-system/sw/bin/zsh",
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
				Root("/", "ebf083d1b175911782d413369b64ce7c", container.BindWritable).
				Proc("/proc").
				Tmpfs(hst.Tmp, 4096, 0755).
				Dev("/dev").Mqueue("/dev/mqueue").
				Bind("/dev/dri", "/dev/dri", container.BindWritable|container.BindDevice|container.BindOptional).
				Bind("/dev/kvm", "/dev/kvm", container.BindWritable|container.BindDevice|container.BindOptional).
				Readonly("/var/run/nscd", 0755).
				Tmpfs("/run/user/1971", 8192, 0755).
				Tmpfs("/run/dbus", 8192, 0755).
				Etc("/etc", "ebf083d1b175911782d413369b64ce7c").
				Tmpfs("/run/user", 4096, 0755).
				Bind("/tmp/hakurei.1971/runtime/9", "/run/user/65534", container.BindWritable).
				Bind("/tmp/hakurei.1971/tmpdir/9", "/tmp", container.BindWritable).
				Bind("/home/chronos", "/home/chronos", container.BindWritable).
				Place("/etc/passwd", []byte("chronos:x:65534:65534:Hakurei:/home/chronos:/run/current-system/sw/bin/zsh\n")).
				Place("/etc/group", []byte("hakurei:x:65534:\n")).
				Bind("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland", "/run/user/65534/wayland-0", 0).
				Bind("/run/user/1971/hakurei/ebf083d1b175911782d413369b64ce7c/pulse", "/run/user/65534/pulse/native", 0).
				Place(hst.Tmp+"/pulse-cookie", nil).
				Bind("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/bus", "/run/user/65534/bus", 0).
				Bind("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/system_bus_socket", "/run/dbus/system_bus_socket", 0),
			SeccompPresets: seccomp.PresetExt | seccomp.PresetDenyDevel,
			HostNet:        true,
			RetainSession:  true,
			ForwardCancel:  true,
		},
	},
}
