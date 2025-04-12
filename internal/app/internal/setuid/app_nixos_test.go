package setuid_test

import (
	"git.gensokyo.uk/security/fortify/acl"
	"git.gensokyo.uk/security/fortify/dbus"
	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/internal/app"
	"git.gensokyo.uk/security/fortify/sandbox"
	"git.gensokyo.uk/security/fortify/system"
)

var testCasesNixos = []sealTestCase{
	{
		"nixos chromium direct wayland", new(stubNixOS),
		&fst.Config{
			ID:          "org.chromium.Chromium",
			Path:        "/nix/store/yqivzpzzn7z5x0lq9hmbzygh45d8rhqd-chromium-start",
			Enablements: system.EWayland | system.EDBus | system.EPulse,

			Container: &fst.ContainerConfig{
				Userns: true, Net: true, MapRealUID: true, Env: nil, AutoEtc: true,
				Filesystem: []*fst.FilesystemConfig{
					{Src: "/bin", Must: true}, {Src: "/usr/bin", Must: true},
					{Src: "/nix/store", Must: true}, {Src: "/run/current-system", Must: true},
					{Src: "/sys/block"}, {Src: "/sys/bus"}, {Src: "/sys/class"}, {Src: "/sys/dev"}, {Src: "/sys/devices"},
					{Src: "/run/opengl-driver", Must: true}, {Src: "/dev/dri", Device: true},
				},
				Cover: []string{"/var/run/nscd"},
			},
			SystemBus: &dbus.Config{
				Talk:   []string{"org.bluez", "org.freedesktop.Avahi", "org.freedesktop.UPower"},
				Filter: true,
			},
			SessionBus: &dbus.Config{
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

			Username: "u0_a1",
			Data:     "/var/lib/persist/module/fortify/0/1",
			Identity: 1, Groups: []string{},
		},
		app.ID{
			0x8e, 0x2c, 0x76, 0xb0,
			0x66, 0xda, 0xbe, 0x57,
			0x4c, 0xf0, 0x73, 0xbd,
			0xb4, 0x6e, 0xb5, 0xc1,
		},
		system.New(1000001).
			Ensure("/tmp/fortify.1971", 0711).
			Ensure("/tmp/fortify.1971/tmpdir", 0700).UpdatePermType(system.User, "/tmp/fortify.1971/tmpdir", acl.Execute).
			Ensure("/tmp/fortify.1971/tmpdir/1", 01700).UpdatePermType(system.User, "/tmp/fortify.1971/tmpdir/1", acl.Read, acl.Write, acl.Execute).
			Ensure("/run/user/1971/fortify", 0700).UpdatePermType(system.User, "/run/user/1971/fortify", acl.Execute).
			Ensure("/run/user/1971", 0700).UpdatePermType(system.User, "/run/user/1971", acl.Execute). // this is ordered as is because the previous Ensure only calls mkdir if XDG_RUNTIME_DIR is unset
			UpdatePermType(system.EWayland, "/run/user/1971/wayland-0", acl.Read, acl.Write, acl.Execute).
			Ephemeral(system.Process, "/run/user/1971/fortify/8e2c76b066dabe574cf073bdb46eb5c1", 0700).UpdatePermType(system.Process, "/run/user/1971/fortify/8e2c76b066dabe574cf073bdb46eb5c1", acl.Execute).
			Link("/run/user/1971/pulse/native", "/run/user/1971/fortify/8e2c76b066dabe574cf073bdb46eb5c1/pulse").
			CopyFile(nil, "/home/ophestra/xdg/config/pulse/cookie", 256, 256).
			Ephemeral(system.Process, "/tmp/fortify.1971/8e2c76b066dabe574cf073bdb46eb5c1", 0711).
			MustProxyDBus("/tmp/fortify.1971/8e2c76b066dabe574cf073bdb46eb5c1/bus", &dbus.Config{
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
			}, "/tmp/fortify.1971/8e2c76b066dabe574cf073bdb46eb5c1/system_bus_socket", &dbus.Config{
				Talk: []string{
					"org.bluez",
					"org.freedesktop.Avahi",
					"org.freedesktop.UPower",
				},
				Filter: true,
			}).
			UpdatePerm("/tmp/fortify.1971/8e2c76b066dabe574cf073bdb46eb5c1/bus", acl.Read, acl.Write).
			UpdatePerm("/tmp/fortify.1971/8e2c76b066dabe574cf073bdb46eb5c1/system_bus_socket", acl.Read, acl.Write),
		&sandbox.Params{
			Uid:   1971,
			Gid:   100,
			Flags: sandbox.FAllowNet | sandbox.FAllowUserns,
			Dir:   "/var/lib/persist/module/fortify/0/1",
			Path:  "/nix/store/yqivzpzzn7z5x0lq9hmbzygh45d8rhqd-chromium-start",
			Args:  []string{"/nix/store/yqivzpzzn7z5x0lq9hmbzygh45d8rhqd-chromium-start"},
			Env: []string{
				"DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1971/bus",
				"DBUS_SYSTEM_BUS_ADDRESS=unix:path=/run/dbus/system_bus_socket",
				"HOME=/var/lib/persist/module/fortify/0/1",
				"PULSE_COOKIE=" + fst.Tmp + "/pulse-cookie",
				"PULSE_SERVER=unix:/run/user/1971/pulse/native",
				"SHELL=/run/current-system/sw/bin/zsh",
				"TERM=xterm-256color",
				"USER=u0_a1",
				"WAYLAND_DISPLAY=wayland-0",
				"XDG_RUNTIME_DIR=/run/user/1971",
				"XDG_SESSION_CLASS=user",
				"XDG_SESSION_TYPE=tty",
			},
			Ops: new(sandbox.Ops).
				Proc("/proc").
				Tmpfs(fst.Tmp, 4096, 0755).
				Dev("/dev").Mqueue("/dev/mqueue").
				Bind("/bin", "/bin", 0).
				Bind("/usr/bin", "/usr/bin", 0).
				Bind("/nix/store", "/nix/store", 0).
				Bind("/run/current-system", "/run/current-system", 0).
				Bind("/sys/block", "/sys/block", sandbox.BindOptional).
				Bind("/sys/bus", "/sys/bus", sandbox.BindOptional).
				Bind("/sys/class", "/sys/class", sandbox.BindOptional).
				Bind("/sys/dev", "/sys/dev", sandbox.BindOptional).
				Bind("/sys/devices", "/sys/devices", sandbox.BindOptional).
				Bind("/run/opengl-driver", "/run/opengl-driver", 0).
				Bind("/dev/dri", "/dev/dri", sandbox.BindDevice|sandbox.BindWritable|sandbox.BindOptional).
				Etc("/etc", "8e2c76b066dabe574cf073bdb46eb5c1").
				Tmpfs("/run/user", 4096, 0755).
				Tmpfs("/run/user/1971", 8388608, 0700).
				Bind("/tmp/fortify.1971/tmpdir/1", "/tmp", sandbox.BindWritable).
				Bind("/var/lib/persist/module/fortify/0/1", "/var/lib/persist/module/fortify/0/1", sandbox.BindWritable).
				Place("/etc/passwd", []byte("u0_a1:x:1971:100:Fortify:/var/lib/persist/module/fortify/0/1:/run/current-system/sw/bin/zsh\n")).
				Place("/etc/group", []byte("fortify:x:100:\n")).
				Bind("/run/user/1971/wayland-0", "/run/user/1971/wayland-0", 0).
				Bind("/run/user/1971/fortify/8e2c76b066dabe574cf073bdb46eb5c1/pulse", "/run/user/1971/pulse/native", 0).
				Place(fst.Tmp+"/pulse-cookie", nil).
				Bind("/tmp/fortify.1971/8e2c76b066dabe574cf073bdb46eb5c1/bus", "/run/user/1971/bus", 0).
				Bind("/tmp/fortify.1971/8e2c76b066dabe574cf073bdb46eb5c1/system_bus_socket", "/run/dbus/system_bus_socket", 0).
				Tmpfs("/var/run/nscd", 8192, 0755),
		},
	},
}
