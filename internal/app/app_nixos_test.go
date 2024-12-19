package app_test

import (
	"git.gensokyo.uk/security/fortify/acl"
	"git.gensokyo.uk/security/fortify/dbus"
	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/helper/bwrap"
	"git.gensokyo.uk/security/fortify/internal/system"
)

var testCasesNixos = []sealTestCase{
	{
		"nixos chromium direct wayland", new(stubNixOS),
		&fst.Config{
			ID:      "org.chromium.Chromium",
			Command: []string{"/nix/store/yqivzpzzn7z5x0lq9hmbzygh45d8rhqd-chromium-start"},
			Confinement: fst.ConfinementConfig{
				AppID: 1, Groups: []string{}, Username: "u0_a1",
				Outer: "/var/lib/persist/module/fortify/0/1",
				Sandbox: &fst.SandboxConfig{
					UserNS: true, Net: true, MapRealUID: true, DirectWayland: true, Env: nil,
					Filesystem: []*fst.FilesystemConfig{
						{Src: "/bin", Must: true}, {Src: "/usr/bin", Must: true},
						{Src: "/nix/store", Must: true}, {Src: "/run/current-system", Must: true},
						{Src: "/sys/block"}, {Src: "/sys/bus"}, {Src: "/sys/class"}, {Src: "/sys/dev"}, {Src: "/sys/devices"},
						{Src: "/run/opengl-driver", Must: true}, {Src: "/dev/dri", Device: true},
					}, AutoEtc: true,
					Override: []string{"/var/run/nscd"},
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
				Enablements: system.EWayland.Mask() | system.EDBus.Mask() | system.EPulse.Mask(),
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
			Ephemeral(system.Process, "/tmp/fortify.1971/8e2c76b066dabe574cf073bdb46eb5c1", 0711).
			Ensure("/tmp/fortify.1971/tmpdir", 0700).UpdatePermType(system.User, "/tmp/fortify.1971/tmpdir", acl.Execute).
			Ensure("/tmp/fortify.1971/tmpdir/1", 01700).UpdatePermType(system.User, "/tmp/fortify.1971/tmpdir/1", acl.Read, acl.Write, acl.Execute).
			Ensure("/run/user/1971/fortify", 0700).UpdatePermType(system.User, "/run/user/1971/fortify", acl.Execute).
			Ensure("/run/user/1971", 0700).UpdatePermType(system.User, "/run/user/1971", acl.Execute). // this is ordered as is because the previous Ensure only calls mkdir if XDG_RUNTIME_DIR is unset
			Ephemeral(system.Process, "/run/user/1971/fortify/8e2c76b066dabe574cf073bdb46eb5c1", 0700).UpdatePermType(system.Process, "/run/user/1971/fortify/8e2c76b066dabe574cf073bdb46eb5c1", acl.Execute).
			WriteType(system.Process, "/tmp/fortify.1971/8e2c76b066dabe574cf073bdb46eb5c1/passwd", "u0_a1:x:1971:1971:Fortify:/var/lib/persist/module/fortify/0/1:/run/current-system/sw/bin/zsh\n").
			WriteType(system.Process, "/tmp/fortify.1971/8e2c76b066dabe574cf073bdb46eb5c1/group", "fortify:x:1971:\n").
			Link("/run/user/1971/wayland-0", "/run/user/1971/fortify/8e2c76b066dabe574cf073bdb46eb5c1/wayland").
			UpdatePermType(system.EWayland, "/run/user/1971/wayland-0", acl.Read, acl.Write, acl.Execute).
			Link("/run/user/1971/pulse/native", "/run/user/1971/fortify/8e2c76b066dabe574cf073bdb46eb5c1/pulse").
			CopyFile("/tmp/fortify.1971/8e2c76b066dabe574cf073bdb46eb5c1/pulse-cookie", "/home/ophestra/xdg/config/pulse/cookie").
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
		(&bwrap.Config{
			Net:      true,
			UserNS:   true,
			Chdir:    "/var/lib/persist/module/fortify/0/1",
			Clearenv: true,
			SetEnv: map[string]string{
				"DBUS_SESSION_BUS_ADDRESS": "unix:path=/run/user/1971/bus",
				"DBUS_SYSTEM_BUS_ADDRESS":  "unix:path=/run/dbus/system_bus_socket",
				"HOME":                     "/var/lib/persist/module/fortify/0/1",
				"PULSE_COOKIE":             "/tmp/fortify.1971/8e2c76b066dabe574cf073bdb46eb5c1/pulse-cookie",
				"PULSE_SERVER":             "unix:/run/user/1971/pulse/native",
				"SHELL":                    "/run/current-system/sw/bin/zsh",
				"TERM":                     "xterm-256color",
				"USER":                     "u0_a1",
				"WAYLAND_DISPLAY":          "/run/user/1971/wayland-0",
				"XDG_RUNTIME_DIR":          "/run/user/1971",
				"XDG_SESSION_CLASS":        "user",
				"XDG_SESSION_TYPE":         "tty",
			},
			Chmod:         make(bwrap.ChmodConfig),
			NewSession:    true,
			DieWithParent: true,
			AsInit:        true,
		}).SetUID(1971).SetGID(1971).
			Procfs("/proc").
			Tmpfs("/fortify", 4096).
			DevTmpfs("/dev").Mqueue("/dev/mqueue").
			Bind("/bin", "/bin").
			Bind("/usr/bin", "/usr/bin").
			Bind("/nix/store", "/nix/store").
			Bind("/run/current-system", "/run/current-system").
			Bind("/sys/block", "/sys/block", true).
			Bind("/sys/bus", "/sys/bus", true).
			Bind("/sys/class", "/sys/class", true).
			Bind("/sys/dev", "/sys/dev", true).
			Bind("/sys/devices", "/sys/devices", true).
			Bind("/run/opengl-driver", "/run/opengl-driver").
			Bind("/dev/dri", "/dev/dri", true, true, true).
			Bind("/etc", "/fortify/etc").
			Symlink("/fortify/etc/alsa", "/etc/alsa").
			Symlink("/fortify/etc/bashrc", "/etc/bashrc").
			Symlink("/fortify/etc/binfmt.d", "/etc/binfmt.d").
			Symlink("/fortify/etc/dbus-1", "/etc/dbus-1").
			Symlink("/fortify/etc/default", "/etc/default").
			Symlink("/fortify/etc/ethertypes", "/etc/ethertypes").
			Symlink("/fortify/etc/fonts", "/etc/fonts").
			Symlink("/fortify/etc/fstab", "/etc/fstab").
			Symlink("/fortify/etc/fuse.conf", "/etc/fuse.conf").
			Symlink("/fortify/etc/host.conf", "/etc/host.conf").
			Symlink("/fortify/etc/hostid", "/etc/hostid").
			Symlink("/fortify/etc/hostname", "/etc/hostname").
			Symlink("/fortify/etc/hostname.CHECKSUM", "/etc/hostname.CHECKSUM").
			Symlink("/fortify/etc/hosts", "/etc/hosts").
			Symlink("/fortify/etc/inputrc", "/etc/inputrc").
			Symlink("/fortify/etc/ipsec.d", "/etc/ipsec.d").
			Symlink("/fortify/etc/issue", "/etc/issue").
			Symlink("/fortify/etc/kbd", "/etc/kbd").
			Symlink("/fortify/etc/libblockdev", "/etc/libblockdev").
			Symlink("/fortify/etc/locale.conf", "/etc/locale.conf").
			Symlink("/fortify/etc/localtime", "/etc/localtime").
			Symlink("/fortify/etc/login.defs", "/etc/login.defs").
			Symlink("/fortify/etc/lsb-release", "/etc/lsb-release").
			Symlink("/fortify/etc/lvm", "/etc/lvm").
			Symlink("/fortify/etc/machine-id", "/etc/machine-id").
			Symlink("/fortify/etc/man_db.conf", "/etc/man_db.conf").
			Symlink("/fortify/etc/modprobe.d", "/etc/modprobe.d").
			Symlink("/fortify/etc/modules-load.d", "/etc/modules-load.d").
			Symlink("/proc/mounts", "/etc/mtab").
			Symlink("/fortify/etc/nanorc", "/etc/nanorc").
			Symlink("/fortify/etc/netgroup", "/etc/netgroup").
			Symlink("/fortify/etc/NetworkManager", "/etc/NetworkManager").
			Symlink("/fortify/etc/nix", "/etc/nix").
			Symlink("/fortify/etc/nixos", "/etc/nixos").
			Symlink("/fortify/etc/NIXOS", "/etc/NIXOS").
			Symlink("/fortify/etc/nscd.conf", "/etc/nscd.conf").
			Symlink("/fortify/etc/nsswitch.conf", "/etc/nsswitch.conf").
			Symlink("/fortify/etc/opensnitchd", "/etc/opensnitchd").
			Symlink("/fortify/etc/os-release", "/etc/os-release").
			Symlink("/fortify/etc/pam", "/etc/pam").
			Symlink("/fortify/etc/pam.d", "/etc/pam.d").
			Symlink("/fortify/etc/pipewire", "/etc/pipewire").
			Symlink("/fortify/etc/pki", "/etc/pki").
			Symlink("/fortify/etc/polkit-1", "/etc/polkit-1").
			Symlink("/fortify/etc/profile", "/etc/profile").
			Symlink("/fortify/etc/protocols", "/etc/protocols").
			Symlink("/fortify/etc/qemu", "/etc/qemu").
			Symlink("/fortify/etc/resolv.conf", "/etc/resolv.conf").
			Symlink("/fortify/etc/resolvconf.conf", "/etc/resolvconf.conf").
			Symlink("/fortify/etc/rpc", "/etc/rpc").
			Symlink("/fortify/etc/samba", "/etc/samba").
			Symlink("/fortify/etc/sddm.conf", "/etc/sddm.conf").
			Symlink("/fortify/etc/secureboot", "/etc/secureboot").
			Symlink("/fortify/etc/services", "/etc/services").
			Symlink("/fortify/etc/set-environment", "/etc/set-environment").
			Symlink("/fortify/etc/shadow", "/etc/shadow").
			Symlink("/fortify/etc/shells", "/etc/shells").
			Symlink("/fortify/etc/ssh", "/etc/ssh").
			Symlink("/fortify/etc/ssl", "/etc/ssl").
			Symlink("/fortify/etc/static", "/etc/static").
			Symlink("/fortify/etc/subgid", "/etc/subgid").
			Symlink("/fortify/etc/subuid", "/etc/subuid").
			Symlink("/fortify/etc/sudoers", "/etc/sudoers").
			Symlink("/fortify/etc/sysctl.d", "/etc/sysctl.d").
			Symlink("/fortify/etc/systemd", "/etc/systemd").
			Symlink("/fortify/etc/terminfo", "/etc/terminfo").
			Symlink("/fortify/etc/tmpfiles.d", "/etc/tmpfiles.d").
			Symlink("/fortify/etc/udev", "/etc/udev").
			Symlink("/fortify/etc/udisks2", "/etc/udisks2").
			Symlink("/fortify/etc/UPower", "/etc/UPower").
			Symlink("/fortify/etc/vconsole.conf", "/etc/vconsole.conf").
			Symlink("/fortify/etc/X11", "/etc/X11").
			Symlink("/fortify/etc/zfs", "/etc/zfs").
			Symlink("/fortify/etc/zinputrc", "/etc/zinputrc").
			Symlink("/fortify/etc/zoneinfo", "/etc/zoneinfo").
			Symlink("/fortify/etc/zprofile", "/etc/zprofile").
			Symlink("/fortify/etc/zshenv", "/etc/zshenv").
			Symlink("/fortify/etc/zshrc", "/etc/zshrc").
			Bind("/tmp/fortify.1971/tmpdir/1", "/tmp", false, true).
			Tmpfs("/tmp/fortify.1971", 1048576).
			Tmpfs("/run/user", 1048576).
			Tmpfs("/run/user/1971", 8388608).
			Bind("/var/lib/persist/module/fortify/0/1", "/var/lib/persist/module/fortify/0/1", false, true).
			Bind("/tmp/fortify.1971/8e2c76b066dabe574cf073bdb46eb5c1/passwd", "/etc/passwd").
			Bind("/tmp/fortify.1971/8e2c76b066dabe574cf073bdb46eb5c1/group", "/etc/group").
			Bind("/run/user/1971/fortify/8e2c76b066dabe574cf073bdb46eb5c1/wayland", "/run/user/1971/wayland-0").
			Bind("/run/user/1971/fortify/8e2c76b066dabe574cf073bdb46eb5c1/pulse", "/run/user/1971/pulse/native").
			Bind("/tmp/fortify.1971/8e2c76b066dabe574cf073bdb46eb5c1/pulse-cookie", "/tmp/fortify.1971/8e2c76b066dabe574cf073bdb46eb5c1/pulse-cookie").
			Bind("/tmp/fortify.1971/8e2c76b066dabe574cf073bdb46eb5c1/bus", "/run/user/1971/bus").
			Bind("/tmp/fortify.1971/8e2c76b066dabe574cf073bdb46eb5c1/system_bus_socket", "/run/dbus/system_bus_socket").
			Tmpfs("/var/run/nscd", 8192),
	},
}
