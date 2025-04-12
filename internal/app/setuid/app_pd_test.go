package setuid_test

import (
	"os"

	"git.gensokyo.uk/security/fortify/acl"
	"git.gensokyo.uk/security/fortify/dbus"
	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/sandbox"
	"git.gensokyo.uk/security/fortify/system"
)

var testCasesPd = []sealTestCase{
	{
		"nixos permissive defaults no enablements", new(stubNixOS),
		&fst.Config{
			Confinement: fst.ConfinementConfig{
				AppID:    0,
				Username: "chronos",
				Outer:    "/home/chronos",
			},
		},
		fst.ID{
			0x4a, 0x45, 0x0b, 0x65,
			0x96, 0xd7, 0xbc, 0x15,
			0xbd, 0x01, 0x78, 0x0e,
			0xb9, 0xa6, 0x07, 0xac,
		},
		system.New(1000000).
			Ensure("/tmp/fortify.1971", 0711).
			Ensure("/tmp/fortify.1971/tmpdir", 0700).UpdatePermType(system.User, "/tmp/fortify.1971/tmpdir", acl.Execute).
			Ensure("/tmp/fortify.1971/tmpdir/0", 01700).UpdatePermType(system.User, "/tmp/fortify.1971/tmpdir/0", acl.Read, acl.Write, acl.Execute),
		&sandbox.Params{
			Flags: sandbox.FAllowNet | sandbox.FAllowUserns | sandbox.FAllowTTY,
			Dir:   "/home/chronos",
			Path:  "/run/current-system/sw/bin/zsh",
			Args:  []string{"/run/current-system/sw/bin/zsh"},
			Env: []string{
				"HOME=/home/chronos",
				"SHELL=/run/current-system/sw/bin/zsh",
				"TERM=xterm-256color",
				"USER=chronos",
				"XDG_RUNTIME_DIR=/run/user/65534",
				"XDG_SESSION_CLASS=user",
				"XDG_SESSION_TYPE=tty",
			},
			Ops: new(sandbox.Ops).
				Proc("/proc").
				Tmpfs(fst.Tmp, 4096, 0755).
				Dev("/dev").Mqueue("/dev/mqueue").
				Bind("/bin", "/bin", sandbox.BindWritable).
				Bind("/boot", "/boot", sandbox.BindWritable).
				Bind("/home", "/home", sandbox.BindWritable).
				Bind("/lib", "/lib", sandbox.BindWritable).
				Bind("/lib64", "/lib64", sandbox.BindWritable).
				Bind("/nix", "/nix", sandbox.BindWritable).
				Bind("/root", "/root", sandbox.BindWritable).
				Bind("/run", "/run", sandbox.BindWritable).
				Bind("/srv", "/srv", sandbox.BindWritable).
				Bind("/sys", "/sys", sandbox.BindWritable).
				Bind("/usr", "/usr", sandbox.BindWritable).
				Bind("/var", "/var", sandbox.BindWritable).
				Bind("/dev/kvm", "/dev/kvm", sandbox.BindWritable|sandbox.BindDevice|sandbox.BindOptional).
				Tmpfs("/run/user/1971", 8192, 0755).
				Tmpfs("/run/dbus", 8192, 0755).
				Etc("/etc", "4a450b6596d7bc15bd01780eb9a607ac").
				Tmpfs("/run/user", 4096, 0755).
				Tmpfs("/run/user/65534", 8388608, 0700).
				Bind("/tmp/fortify.1971/tmpdir/0", "/tmp", sandbox.BindWritable).
				Bind("/home/chronos", "/home/chronos", sandbox.BindWritable).
				Place("/etc/passwd", []byte("chronos:x:65534:65534:Fortify:/home/chronos:/run/current-system/sw/bin/zsh\n")).
				Place("/etc/group", []byte("fortify:x:65534:\n")).
				Tmpfs("/var/run/nscd", 8192, 0755),
		},
	},
	{
		"nixos permissive defaults chromium", new(stubNixOS),
		&fst.Config{
			ID:   "org.chromium.Chromium",
			Args: []string{"zsh", "-c", "exec chromium "},
			Confinement: fst.ConfinementConfig{
				AppID:    9,
				Groups:   []string{"video"},
				Username: "chronos",
				Outer:    "/home/chronos",
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
		},
		fst.ID{
			0xeb, 0xf0, 0x83, 0xd1,
			0xb1, 0x75, 0x91, 0x17,
			0x82, 0xd4, 0x13, 0x36,
			0x9b, 0x64, 0xce, 0x7c,
		},
		system.New(1000009).
			Ensure("/tmp/fortify.1971", 0711).
			Ensure("/tmp/fortify.1971/tmpdir", 0700).UpdatePermType(system.User, "/tmp/fortify.1971/tmpdir", acl.Execute).
			Ensure("/tmp/fortify.1971/tmpdir/9", 01700).UpdatePermType(system.User, "/tmp/fortify.1971/tmpdir/9", acl.Read, acl.Write, acl.Execute).
			Ephemeral(system.Process, "/tmp/fortify.1971/ebf083d1b175911782d413369b64ce7c", 0711).
			Wayland(new(*os.File), "/tmp/fortify.1971/ebf083d1b175911782d413369b64ce7c/wayland", "/run/user/1971/wayland-0", "org.chromium.Chromium", "ebf083d1b175911782d413369b64ce7c").
			Ensure("/run/user/1971/fortify", 0700).UpdatePermType(system.User, "/run/user/1971/fortify", acl.Execute).
			Ensure("/run/user/1971", 0700).UpdatePermType(system.User, "/run/user/1971", acl.Execute). // this is ordered as is because the previous Ensure only calls mkdir if XDG_RUNTIME_DIR is unset
			Ephemeral(system.Process, "/run/user/1971/fortify/ebf083d1b175911782d413369b64ce7c", 0700).UpdatePermType(system.Process, "/run/user/1971/fortify/ebf083d1b175911782d413369b64ce7c", acl.Execute).
			Link("/run/user/1971/pulse/native", "/run/user/1971/fortify/ebf083d1b175911782d413369b64ce7c/pulse").
			CopyFile(new([]byte), "/home/ophestra/xdg/config/pulse/cookie", 256, 256).
			MustProxyDBus("/tmp/fortify.1971/ebf083d1b175911782d413369b64ce7c/bus", &dbus.Config{
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
			}, "/tmp/fortify.1971/ebf083d1b175911782d413369b64ce7c/system_bus_socket", &dbus.Config{
				Talk: []string{
					"org.bluez",
					"org.freedesktop.Avahi",
					"org.freedesktop.UPower",
				},
				Filter: true,
			}).
			UpdatePerm("/tmp/fortify.1971/ebf083d1b175911782d413369b64ce7c/bus", acl.Read, acl.Write).
			UpdatePerm("/tmp/fortify.1971/ebf083d1b175911782d413369b64ce7c/system_bus_socket", acl.Read, acl.Write),
		&sandbox.Params{
			Flags: sandbox.FAllowNet | sandbox.FAllowUserns | sandbox.FAllowTTY,
			Dir:   "/home/chronos",
			Path:  "/run/current-system/sw/bin/zsh",
			Args:  []string{"zsh", "-c", "exec chromium "},
			Env: []string{
				"DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/65534/bus",
				"DBUS_SYSTEM_BUS_ADDRESS=unix:path=/run/dbus/system_bus_socket",
				"HOME=/home/chronos",
				"PULSE_COOKIE=" + fst.Tmp + "/pulse-cookie",
				"PULSE_SERVER=unix:/run/user/65534/pulse/native",
				"SHELL=/run/current-system/sw/bin/zsh",
				"TERM=xterm-256color",
				"USER=chronos",
				"WAYLAND_DISPLAY=wayland-0",
				"XDG_RUNTIME_DIR=/run/user/65534",
				"XDG_SESSION_CLASS=user",
				"XDG_SESSION_TYPE=tty",
			},
			Ops: new(sandbox.Ops).
				Proc("/proc").
				Tmpfs(fst.Tmp, 4096, 0755).
				Dev("/dev").Mqueue("/dev/mqueue").
				Bind("/bin", "/bin", sandbox.BindWritable).
				Bind("/boot", "/boot", sandbox.BindWritable).
				Bind("/home", "/home", sandbox.BindWritable).
				Bind("/lib", "/lib", sandbox.BindWritable).
				Bind("/lib64", "/lib64", sandbox.BindWritable).
				Bind("/nix", "/nix", sandbox.BindWritable).
				Bind("/root", "/root", sandbox.BindWritable).
				Bind("/run", "/run", sandbox.BindWritable).
				Bind("/srv", "/srv", sandbox.BindWritable).
				Bind("/sys", "/sys", sandbox.BindWritable).
				Bind("/usr", "/usr", sandbox.BindWritable).
				Bind("/var", "/var", sandbox.BindWritable).
				Bind("/dev/dri", "/dev/dri", sandbox.BindWritable|sandbox.BindDevice|sandbox.BindOptional).
				Bind("/dev/kvm", "/dev/kvm", sandbox.BindWritable|sandbox.BindDevice|sandbox.BindOptional).
				Tmpfs("/run/user/1971", 8192, 0755).
				Tmpfs("/run/dbus", 8192, 0755).
				Etc("/etc", "ebf083d1b175911782d413369b64ce7c").
				Tmpfs("/run/user", 4096, 0755).
				Tmpfs("/run/user/65534", 8388608, 0700).
				Bind("/tmp/fortify.1971/tmpdir/9", "/tmp", sandbox.BindWritable).
				Bind("/home/chronos", "/home/chronos", sandbox.BindWritable).
				Place("/etc/passwd", []byte("chronos:x:65534:65534:Fortify:/home/chronos:/run/current-system/sw/bin/zsh\n")).
				Place("/etc/group", []byte("fortify:x:65534:\n")).
				Bind("/tmp/fortify.1971/ebf083d1b175911782d413369b64ce7c/wayland", "/run/user/65534/wayland-0", 0).
				Bind("/run/user/1971/fortify/ebf083d1b175911782d413369b64ce7c/pulse", "/run/user/65534/pulse/native", 0).
				Place(fst.Tmp+"/pulse-cookie", nil).
				Bind("/tmp/fortify.1971/ebf083d1b175911782d413369b64ce7c/bus", "/run/user/65534/bus", 0).
				Bind("/tmp/fortify.1971/ebf083d1b175911782d413369b64ce7c/system_bus_socket", "/run/dbus/system_bus_socket", 0).
				Tmpfs("/var/run/nscd", 8192, 0755),
		},
	},
}
