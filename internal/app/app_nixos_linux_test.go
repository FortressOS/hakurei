package app_test

import (
	"syscall"

	"hakurei.app/container"
	"hakurei.app/container/seccomp"
	"hakurei.app/hst"
	"hakurei.app/internal/app/state"
	"hakurei.app/system"
	"hakurei.app/system/acl"
	"hakurei.app/system/dbus"
)

func m(pathname string) *container.Absolute { return container.MustAbs(pathname) }
func f(c hst.FilesystemConfig) hst.FilesystemConfigJSON {
	return hst.FilesystemConfigJSON{FilesystemConfig: c}
}

var testCasesNixos = []sealTestCase{
	{
		"nixos chromium direct wayland", new(stubNixOS),
		&hst.Config{
			ID:          "org.chromium.Chromium",
			Path:        m("/nix/store/yqivzpzzn7z5x0lq9hmbzygh45d8rhqd-chromium-start"),
			Enablements: hst.NewEnablements(system.EWayland | system.EDBus | system.EPulse),
			Shell:       m("/run/current-system/sw/bin/zsh"),

			Container: &hst.ContainerConfig{
				Userns: true, Net: true, MapRealUID: true, Env: nil, AutoEtc: true,
				Filesystem: []hst.FilesystemConfigJSON{
					f(&hst.FSBind{Src: m("/bin")}),
					f(&hst.FSBind{Src: m("/usr/bin/")}),
					f(&hst.FSBind{Src: m("/nix/store")}),
					f(&hst.FSBind{Src: m("/run/current-system")}),
					f(&hst.FSBind{Src: m("/sys/block"), Optional: true}),
					f(&hst.FSBind{Src: m("/sys/bus"), Optional: true}),
					f(&hst.FSBind{Src: m("/sys/class"), Optional: true}),
					f(&hst.FSBind{Src: m("/sys/dev"), Optional: true}),
					f(&hst.FSBind{Src: m("/sys/devices"), Optional: true}),
					f(&hst.FSBind{Src: m("/run/opengl-driver")}),
					f(&hst.FSBind{Src: m("/dev/dri"), Device: true, Optional: true}),
				},
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
			Data:     m("/var/lib/persist/module/hakurei/0/1"),
			Identity: 1, Groups: []string{},
		},
		state.ID{
			0x8e, 0x2c, 0x76, 0xb0,
			0x66, 0xda, 0xbe, 0x57,
			0x4c, 0xf0, 0x73, 0xbd,
			0xb4, 0x6e, 0xb5, 0xc1,
		},
		system.New(1000001).
			Ensure("/tmp/hakurei.1971", 0711).
			Ensure("/tmp/hakurei.1971/runtime", 0700).UpdatePermType(system.User, "/tmp/hakurei.1971/runtime", acl.Execute).
			Ensure("/tmp/hakurei.1971/runtime/1", 0700).UpdatePermType(system.User, "/tmp/hakurei.1971/runtime/1", acl.Read, acl.Write, acl.Execute).
			Ensure("/tmp/hakurei.1971/tmpdir", 0700).UpdatePermType(system.User, "/tmp/hakurei.1971/tmpdir", acl.Execute).
			Ensure("/tmp/hakurei.1971/tmpdir/1", 01700).UpdatePermType(system.User, "/tmp/hakurei.1971/tmpdir/1", acl.Read, acl.Write, acl.Execute).
			Ensure("/run/user/1971/hakurei", 0700).UpdatePermType(system.User, "/run/user/1971/hakurei", acl.Execute).
			Ensure("/run/user/1971", 0700).UpdatePermType(system.User, "/run/user/1971", acl.Execute). // this is ordered as is because the previous Ensure only calls mkdir if XDG_RUNTIME_DIR is unset
			UpdatePermType(system.EWayland, "/run/user/1971/wayland-0", acl.Read, acl.Write, acl.Execute).
			Ephemeral(system.Process, "/run/user/1971/hakurei/8e2c76b066dabe574cf073bdb46eb5c1", 0700).UpdatePermType(system.Process, "/run/user/1971/hakurei/8e2c76b066dabe574cf073bdb46eb5c1", acl.Execute).
			Link("/run/user/1971/pulse/native", "/run/user/1971/hakurei/8e2c76b066dabe574cf073bdb46eb5c1/pulse").
			CopyFile(nil, "/home/ophestra/xdg/config/pulse/cookie", 256, 256).
			Ephemeral(system.Process, "/tmp/hakurei.1971/8e2c76b066dabe574cf073bdb46eb5c1", 0711).
			MustProxyDBus("/tmp/hakurei.1971/8e2c76b066dabe574cf073bdb46eb5c1/bus", &dbus.Config{
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
			}, "/tmp/hakurei.1971/8e2c76b066dabe574cf073bdb46eb5c1/system_bus_socket", &dbus.Config{
				Talk: []string{
					"org.bluez",
					"org.freedesktop.Avahi",
					"org.freedesktop.UPower",
				},
				Filter: true,
			}).
			UpdatePerm("/tmp/hakurei.1971/8e2c76b066dabe574cf073bdb46eb5c1/bus", acl.Read, acl.Write).
			UpdatePerm("/tmp/hakurei.1971/8e2c76b066dabe574cf073bdb46eb5c1/system_bus_socket", acl.Read, acl.Write),
		&container.Params{
			Uid:  1971,
			Gid:  100,
			Dir:  m("/var/lib/persist/module/hakurei/0/1"),
			Path: m("/nix/store/yqivzpzzn7z5x0lq9hmbzygh45d8rhqd-chromium-start"),
			Args: []string{"/nix/store/yqivzpzzn7z5x0lq9hmbzygh45d8rhqd-chromium-start"},
			Env: []string{
				"DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1971/bus",
				"DBUS_SYSTEM_BUS_ADDRESS=unix:path=/run/dbus/system_bus_socket",
				"HOME=/var/lib/persist/module/hakurei/0/1",
				"PULSE_COOKIE=" + hst.Tmp + "/pulse-cookie",
				"PULSE_SERVER=unix:/run/user/1971/pulse/native",
				"SHELL=/run/current-system/sw/bin/zsh",
				"TERM=xterm-256color",
				"USER=u0_a1",
				"WAYLAND_DISPLAY=wayland-0",
				"XDG_RUNTIME_DIR=/run/user/1971",
				"XDG_SESSION_CLASS=user",
				"XDG_SESSION_TYPE=tty",
			},
			Ops: new(container.Ops).
				Proc(m("/proc/")).
				Tmpfs(hst.AbsTmp, 4096, 0755).
				DevWritable(m("/dev/"), true).
				Bind(m("/bin"), m("/bin"), 0).
				Bind(m("/usr/bin/"), m("/usr/bin/"), 0).
				Bind(m("/nix/store"), m("/nix/store"), 0).
				Bind(m("/run/current-system"), m("/run/current-system"), 0).
				Bind(m("/sys/block"), m("/sys/block"), container.BindOptional).
				Bind(m("/sys/bus"), m("/sys/bus"), container.BindOptional).
				Bind(m("/sys/class"), m("/sys/class"), container.BindOptional).
				Bind(m("/sys/dev"), m("/sys/dev"), container.BindOptional).
				Bind(m("/sys/devices"), m("/sys/devices"), container.BindOptional).
				Bind(m("/run/opengl-driver"), m("/run/opengl-driver"), 0).
				Bind(m("/dev/dri"), m("/dev/dri"), container.BindDevice|container.BindWritable|container.BindOptional).
				Etc(m("/etc/"), "8e2c76b066dabe574cf073bdb46eb5c1").
				Remount(m("/dev/"), syscall.MS_RDONLY).
				Tmpfs(m("/run/user/"), 4096, 0755).
				Bind(m("/tmp/hakurei.1971/runtime/1"), m("/run/user/1971"), container.BindWritable).
				Bind(m("/tmp/hakurei.1971/tmpdir/1"), m("/tmp/"), container.BindWritable).
				Bind(m("/var/lib/persist/module/hakurei/0/1"), m("/var/lib/persist/module/hakurei/0/1"), container.BindWritable).
				Place(m("/etc/passwd"), []byte("u0_a1:x:1971:100:Hakurei:/var/lib/persist/module/hakurei/0/1:/run/current-system/sw/bin/zsh\n")).
				Place(m("/etc/group"), []byte("hakurei:x:100:\n")).
				Bind(m("/run/user/1971/wayland-0"), m("/run/user/1971/wayland-0"), 0).
				Bind(m("/run/user/1971/hakurei/8e2c76b066dabe574cf073bdb46eb5c1/pulse"), m("/run/user/1971/pulse/native"), 0).
				Place(m(hst.Tmp+"/pulse-cookie"), nil).
				Bind(m("/tmp/hakurei.1971/8e2c76b066dabe574cf073bdb46eb5c1/bus"), m("/run/user/1971/bus"), 0).
				Bind(m("/tmp/hakurei.1971/8e2c76b066dabe574cf073bdb46eb5c1/system_bus_socket"), m("/run/dbus/system_bus_socket"), 0).
				Remount(m("/"), syscall.MS_RDONLY),
			SeccompPresets: seccomp.PresetExt | seccomp.PresetDenyTTY | seccomp.PresetDenyDevel,
			HostNet:        true,
			ForwardCancel:  true,
		},
	},
}
