package app_test

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
			Ensure("/run/user/1971/fortify", 0700).UpdatePermType(system.User, "/run/user/1971/fortify", acl.Execute).
			Ensure("/run/user/1971", 0700).UpdatePermType(system.User, "/run/user/1971", acl.Execute). // this is ordered as is because the previous Ensure only calls mkdir if XDG_RUNTIME_DIR is unset
			Ephemeral(system.Process, "/tmp/fortify.1971/4a450b6596d7bc15bd01780eb9a607ac", 0711).
			Ephemeral(system.Process, "/run/user/1971/fortify/4a450b6596d7bc15bd01780eb9a607ac", 0700).UpdatePermType(system.Process, "/run/user/1971/fortify/4a450b6596d7bc15bd01780eb9a607ac", acl.Execute).
			Ensure("/tmp/fortify.1971/tmpdir", 0700).UpdatePermType(system.User, "/tmp/fortify.1971/tmpdir", acl.Execute).
			Ensure("/tmp/fortify.1971/tmpdir/0", 01700).UpdatePermType(system.User, "/tmp/fortify.1971/tmpdir/0", acl.Read, acl.Write, acl.Execute),
		&sandbox.Params{
			Flags: sandbox.FAllowNet | sandbox.FAllowUserns | sandbox.FAllowTTY,
			Dir:   "/home/chronos",
			Path:  "/run/current-system/sw/bin/zsh",
			Args:  []string{"/run/current-system/sw/bin/zsh"},
			Env: []string{
				"HOME=/home/chronos",
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
				Tmpfs("/run/user/65534", 8388608, 0755).
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
				Enablements: system.EWayland.Mask() | system.EDBus.Mask() | system.EPulse.Mask(),
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
			Ensure("/run/user/1971/fortify", 0700).UpdatePermType(system.User, "/run/user/1971/fortify", acl.Execute).
			Ensure("/run/user/1971", 0700).UpdatePermType(system.User, "/run/user/1971", acl.Execute). // this is ordered as is because the previous Ensure only calls mkdir if XDG_RUNTIME_DIR is unset
			Ephemeral(system.Process, "/tmp/fortify.1971/ebf083d1b175911782d413369b64ce7c", 0711).
			Ephemeral(system.Process, "/run/user/1971/fortify/ebf083d1b175911782d413369b64ce7c", 0700).UpdatePermType(system.Process, "/run/user/1971/fortify/ebf083d1b175911782d413369b64ce7c", acl.Execute).
			Ensure("/tmp/fortify.1971/tmpdir", 0700).UpdatePermType(system.User, "/tmp/fortify.1971/tmpdir", acl.Execute).
			Ensure("/tmp/fortify.1971/tmpdir/9", 01700).UpdatePermType(system.User, "/tmp/fortify.1971/tmpdir/9", acl.Read, acl.Write, acl.Execute).
			Ensure("/tmp/fortify.1971/wayland", 0711).
			Wayland(new(*os.File), "/tmp/fortify.1971/wayland/ebf083d1b175911782d413369b64ce7c", "/run/user/1971/wayland-0", "org.chromium.Chromium", "ebf083d1b175911782d413369b64ce7c").
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
				Tmpfs("/run/user/65534", 8388608, 0755).
				Bind("/tmp/fortify.1971/tmpdir/9", "/tmp", sandbox.BindWritable).
				Bind("/home/chronos", "/home/chronos", sandbox.BindWritable).
				Place("/etc/passwd", []byte("chronos:x:65534:65534:Fortify:/home/chronos:/run/current-system/sw/bin/zsh\n")).
				Place("/etc/group", []byte("fortify:x:65534:\n")).
				Bind("/tmp/fortify.1971/wayland/ebf083d1b175911782d413369b64ce7c", "/run/user/65534/wayland-0", 0).
				Bind("/run/user/1971/fortify/ebf083d1b175911782d413369b64ce7c/pulse", "/run/user/65534/pulse/native", 0).
				Place(fst.Tmp+"/pulse-cookie", nil).
				Bind("/tmp/fortify.1971/ebf083d1b175911782d413369b64ce7c/bus", "/run/user/65534/bus", 0).
				Bind("/tmp/fortify.1971/ebf083d1b175911782d413369b64ce7c/system_bus_socket", "/run/dbus/system_bus_socket", 0).
				Tmpfs("/var/run/nscd", 8192, 0755),
		},
	},
}
