package app_test

import (
	"git.gensokyo.uk/security/fortify/acl"
	"git.gensokyo.uk/security/fortify/dbus"
	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/sandbox"
	"git.gensokyo.uk/security/fortify/system"
)

var testCasesNixos = []sealTestCase{
	{
		"nixos chromium direct wayland", new(stubNixOS),
		&fst.Config{
			ID:   "org.chromium.Chromium",
			Path: "/nix/store/yqivzpzzn7z5x0lq9hmbzygh45d8rhqd-chromium-start",
			Confinement: fst.ConfinementConfig{
				AppID: 1, Groups: []string{}, Username: "u0_a1",
				Outer: "/var/lib/persist/module/fortify/0/1",
				Sandbox: &fst.SandboxConfig{
					Userns: true, Net: true, MapRealUID: true, DirectWayland: true, Env: nil, AutoEtc: true,
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
				Enablements: system.EWayland | system.EDBus | system.EPulse,
			},
		},
		fst.ID{
			0x8e, 0x2c, 0x76, 0xb0,
			0x66, 0xda, 0xbe, 0x57,
			0x4c, 0xf0, 0x73, 0xbd,
			0xb4, 0x6e, 0xb5, 0xc1,
		},
		system.New(1000001).
			Ensure("/tmp/fortify.1971", 0711).
			Ensure("/run/user/1971/fortify", 0700).UpdatePermType(system.User, "/run/user/1971/fortify", acl.Execute).
			Ensure("/run/user/1971", 0700).UpdatePermType(system.User, "/run/user/1971", acl.Execute). // this is ordered as is because the previous Ensure only calls mkdir if XDG_RUNTIME_DIR is unset
			Ephemeral(system.Process, "/tmp/fortify.1971/8e2c76b066dabe574cf073bdb46eb5c1", 0711).
			Ephemeral(system.Process, "/run/user/1971/fortify/8e2c76b066dabe574cf073bdb46eb5c1", 0700).UpdatePermType(system.Process, "/run/user/1971/fortify/8e2c76b066dabe574cf073bdb46eb5c1", acl.Execute).
			Ensure("/tmp/fortify.1971/tmpdir", 0700).UpdatePermType(system.User, "/tmp/fortify.1971/tmpdir", acl.Execute).
			Ensure("/tmp/fortify.1971/tmpdir/1", 01700).UpdatePermType(system.User, "/tmp/fortify.1971/tmpdir/1", acl.Read, acl.Write, acl.Execute).
			UpdatePermType(system.EWayland, "/run/user/1971/wayland-0", acl.Read, acl.Write, acl.Execute).
			Link("/run/user/1971/pulse/native", "/run/user/1971/fortify/8e2c76b066dabe574cf073bdb46eb5c1/pulse").
			CopyFile(nil, "/home/ophestra/xdg/config/pulse/cookie", 256, 256).
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
				Bind("/etc", fst.Tmp+"/etc", 0).
				Mkdir("/etc", 0700).
				Link(fst.Tmp+"/etc/alsa", "/etc/alsa").
				Link(fst.Tmp+"/etc/bashrc", "/etc/bashrc").
				Link(fst.Tmp+"/etc/binfmt.d", "/etc/binfmt.d").
				Link(fst.Tmp+"/etc/dbus-1", "/etc/dbus-1").
				Link(fst.Tmp+"/etc/default", "/etc/default").
				Link(fst.Tmp+"/etc/ethertypes", "/etc/ethertypes").
				Link(fst.Tmp+"/etc/fonts", "/etc/fonts").
				Link(fst.Tmp+"/etc/fstab", "/etc/fstab").
				Link(fst.Tmp+"/etc/fuse.conf", "/etc/fuse.conf").
				Link(fst.Tmp+"/etc/host.conf", "/etc/host.conf").
				Link(fst.Tmp+"/etc/hostid", "/etc/hostid").
				Link(fst.Tmp+"/etc/hostname", "/etc/hostname").
				Link(fst.Tmp+"/etc/hostname.CHECKSUM", "/etc/hostname.CHECKSUM").
				Link(fst.Tmp+"/etc/hosts", "/etc/hosts").
				Link(fst.Tmp+"/etc/inputrc", "/etc/inputrc").
				Link(fst.Tmp+"/etc/ipsec.d", "/etc/ipsec.d").
				Link(fst.Tmp+"/etc/issue", "/etc/issue").
				Link(fst.Tmp+"/etc/kbd", "/etc/kbd").
				Link(fst.Tmp+"/etc/libblockdev", "/etc/libblockdev").
				Link(fst.Tmp+"/etc/locale.conf", "/etc/locale.conf").
				Link(fst.Tmp+"/etc/localtime", "/etc/localtime").
				Link(fst.Tmp+"/etc/login.defs", "/etc/login.defs").
				Link(fst.Tmp+"/etc/lsb-release", "/etc/lsb-release").
				Link(fst.Tmp+"/etc/lvm", "/etc/lvm").
				Link(fst.Tmp+"/etc/machine-id", "/etc/machine-id").
				Link(fst.Tmp+"/etc/man_db.conf", "/etc/man_db.conf").
				Link(fst.Tmp+"/etc/modprobe.d", "/etc/modprobe.d").
				Link(fst.Tmp+"/etc/modules-load.d", "/etc/modules-load.d").
				Link("/proc/mounts", "/etc/mtab").
				Link(fst.Tmp+"/etc/nanorc", "/etc/nanorc").
				Link(fst.Tmp+"/etc/netgroup", "/etc/netgroup").
				Link(fst.Tmp+"/etc/NetworkManager", "/etc/NetworkManager").
				Link(fst.Tmp+"/etc/nix", "/etc/nix").
				Link(fst.Tmp+"/etc/nixos", "/etc/nixos").
				Link(fst.Tmp+"/etc/NIXOS", "/etc/NIXOS").
				Link(fst.Tmp+"/etc/nscd.conf", "/etc/nscd.conf").
				Link(fst.Tmp+"/etc/nsswitch.conf", "/etc/nsswitch.conf").
				Link(fst.Tmp+"/etc/opensnitchd", "/etc/opensnitchd").
				Link(fst.Tmp+"/etc/os-release", "/etc/os-release").
				Link(fst.Tmp+"/etc/pam", "/etc/pam").
				Link(fst.Tmp+"/etc/pam.d", "/etc/pam.d").
				Link(fst.Tmp+"/etc/pipewire", "/etc/pipewire").
				Link(fst.Tmp+"/etc/pki", "/etc/pki").
				Link(fst.Tmp+"/etc/polkit-1", "/etc/polkit-1").
				Link(fst.Tmp+"/etc/profile", "/etc/profile").
				Link(fst.Tmp+"/etc/protocols", "/etc/protocols").
				Link(fst.Tmp+"/etc/qemu", "/etc/qemu").
				Link(fst.Tmp+"/etc/resolv.conf", "/etc/resolv.conf").
				Link(fst.Tmp+"/etc/resolvconf.conf", "/etc/resolvconf.conf").
				Link(fst.Tmp+"/etc/rpc", "/etc/rpc").
				Link(fst.Tmp+"/etc/samba", "/etc/samba").
				Link(fst.Tmp+"/etc/sddm.conf", "/etc/sddm.conf").
				Link(fst.Tmp+"/etc/secureboot", "/etc/secureboot").
				Link(fst.Tmp+"/etc/services", "/etc/services").
				Link(fst.Tmp+"/etc/set-environment", "/etc/set-environment").
				Link(fst.Tmp+"/etc/shadow", "/etc/shadow").
				Link(fst.Tmp+"/etc/shells", "/etc/shells").
				Link(fst.Tmp+"/etc/ssh", "/etc/ssh").
				Link(fst.Tmp+"/etc/ssl", "/etc/ssl").
				Link(fst.Tmp+"/etc/static", "/etc/static").
				Link(fst.Tmp+"/etc/subgid", "/etc/subgid").
				Link(fst.Tmp+"/etc/subuid", "/etc/subuid").
				Link(fst.Tmp+"/etc/sudoers", "/etc/sudoers").
				Link(fst.Tmp+"/etc/sysctl.d", "/etc/sysctl.d").
				Link(fst.Tmp+"/etc/systemd", "/etc/systemd").
				Link(fst.Tmp+"/etc/terminfo", "/etc/terminfo").
				Link(fst.Tmp+"/etc/tmpfiles.d", "/etc/tmpfiles.d").
				Link(fst.Tmp+"/etc/udev", "/etc/udev").
				Link(fst.Tmp+"/etc/udisks2", "/etc/udisks2").
				Link(fst.Tmp+"/etc/UPower", "/etc/UPower").
				Link(fst.Tmp+"/etc/vconsole.conf", "/etc/vconsole.conf").
				Link(fst.Tmp+"/etc/X11", "/etc/X11").
				Link(fst.Tmp+"/etc/zfs", "/etc/zfs").
				Link(fst.Tmp+"/etc/zinputrc", "/etc/zinputrc").
				Link(fst.Tmp+"/etc/zoneinfo", "/etc/zoneinfo").
				Link(fst.Tmp+"/etc/zprofile", "/etc/zprofile").
				Link(fst.Tmp+"/etc/zshenv", "/etc/zshenv").
				Link(fst.Tmp+"/etc/zshrc", "/etc/zshrc").
				Tmpfs("/run/user", 4096, 0755).
				Tmpfs("/run/user/1971", 8388608, 0755).
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
