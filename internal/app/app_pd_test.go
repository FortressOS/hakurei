package app_test

import (
	"git.gensokyo.uk/security/fortify/acl"
	"git.gensokyo.uk/security/fortify/dbus"
	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/helper/bwrap"
	"git.gensokyo.uk/security/fortify/internal/system"
)

var testCasesPd = []sealTestCase{
	{
		"nixos permissive defaults no enablements", new(stubNixOS),
		&fst.Config{
			Command: make([]string, 0),
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
			Ephemeral(system.Process, "/tmp/fortify.1971/4a450b6596d7bc15bd01780eb9a607ac", 0711).
			Ensure("/tmp/fortify.1971/tmpdir", 0700).UpdatePermType(system.User, "/tmp/fortify.1971/tmpdir", acl.Execute).
			Ensure("/tmp/fortify.1971/tmpdir/0", 01700).UpdatePermType(system.User, "/tmp/fortify.1971/tmpdir/0", acl.Read, acl.Write, acl.Execute).
			Ensure("/run/user/1971/fortify", 0700).UpdatePermType(system.User, "/run/user/1971/fortify", acl.Execute).
			Ensure("/run/user/1971", 0700).UpdatePermType(system.User, "/run/user/1971", acl.Execute). // this is ordered as is because the previous Ensure only calls mkdir if XDG_RUNTIME_DIR is unset
			Ephemeral(system.Process, "/run/user/1971/fortify/4a450b6596d7bc15bd01780eb9a607ac", 0700).UpdatePermType(system.Process, "/run/user/1971/fortify/4a450b6596d7bc15bd01780eb9a607ac", acl.Execute),
		(&bwrap.Config{
			Net:      true,
			UserNS:   true,
			Clearenv: true,
			Syscall:  new(bwrap.SyscallPolicy),
			Chdir:    "/home/chronos",
			SetEnv: map[string]string{
				"HOME":              "/home/chronos",
				"SHELL":             "/run/current-system/sw/bin/zsh",
				"TERM":              "xterm-256color",
				"USER":              "chronos",
				"XDG_RUNTIME_DIR":   "/run/user/65534",
				"XDG_SESSION_CLASS": "user",
				"XDG_SESSION_TYPE":  "tty"},
			Chmod:         make(bwrap.ChmodConfig),
			DieWithParent: true,
			AsInit:        true,
		}).SetUID(65534).SetGID(65534).
			Procfs("/proc").
			Tmpfs(fst.Tmp, 4096).
			DevTmpfs("/dev").Mqueue("/dev/mqueue").
			Bind("/bin", "/bin", false, true).
			Bind("/boot", "/boot", false, true).
			Bind("/home", "/home", false, true).
			Bind("/lib", "/lib", false, true).
			Bind("/lib64", "/lib64", false, true).
			Bind("/nix", "/nix", false, true).
			Bind("/root", "/root", false, true).
			Bind("/run", "/run", false, true).
			Bind("/srv", "/srv", false, true).
			Bind("/sys", "/sys", false, true).
			Bind("/usr", "/usr", false, true).
			Bind("/var", "/var", false, true).
			Bind("/dev/kvm", "/dev/kvm", true, true, true).
			Tmpfs("/run/user/1971", 8192).
			Tmpfs("/run/dbus", 8192).
			Bind("/etc", fst.Tmp+"/etc").
			Symlink(fst.Tmp+"/etc/alsa", "/etc/alsa").
			Symlink(fst.Tmp+"/etc/bashrc", "/etc/bashrc").
			Symlink(fst.Tmp+"/etc/binfmt.d", "/etc/binfmt.d").
			Symlink(fst.Tmp+"/etc/dbus-1", "/etc/dbus-1").
			Symlink(fst.Tmp+"/etc/default", "/etc/default").
			Symlink(fst.Tmp+"/etc/ethertypes", "/etc/ethertypes").
			Symlink(fst.Tmp+"/etc/fonts", "/etc/fonts").
			Symlink(fst.Tmp+"/etc/fstab", "/etc/fstab").
			Symlink(fst.Tmp+"/etc/fuse.conf", "/etc/fuse.conf").
			Symlink(fst.Tmp+"/etc/host.conf", "/etc/host.conf").
			Symlink(fst.Tmp+"/etc/hostid", "/etc/hostid").
			Symlink(fst.Tmp+"/etc/hostname", "/etc/hostname").
			Symlink(fst.Tmp+"/etc/hostname.CHECKSUM", "/etc/hostname.CHECKSUM").
			Symlink(fst.Tmp+"/etc/hosts", "/etc/hosts").
			Symlink(fst.Tmp+"/etc/inputrc", "/etc/inputrc").
			Symlink(fst.Tmp+"/etc/ipsec.d", "/etc/ipsec.d").
			Symlink(fst.Tmp+"/etc/issue", "/etc/issue").
			Symlink(fst.Tmp+"/etc/kbd", "/etc/kbd").
			Symlink(fst.Tmp+"/etc/libblockdev", "/etc/libblockdev").
			Symlink(fst.Tmp+"/etc/locale.conf", "/etc/locale.conf").
			Symlink(fst.Tmp+"/etc/localtime", "/etc/localtime").
			Symlink(fst.Tmp+"/etc/login.defs", "/etc/login.defs").
			Symlink(fst.Tmp+"/etc/lsb-release", "/etc/lsb-release").
			Symlink(fst.Tmp+"/etc/lvm", "/etc/lvm").
			Symlink(fst.Tmp+"/etc/machine-id", "/etc/machine-id").
			Symlink(fst.Tmp+"/etc/man_db.conf", "/etc/man_db.conf").
			Symlink(fst.Tmp+"/etc/modprobe.d", "/etc/modprobe.d").
			Symlink(fst.Tmp+"/etc/modules-load.d", "/etc/modules-load.d").
			Symlink("/proc/mounts", "/etc/mtab").
			Symlink(fst.Tmp+"/etc/nanorc", "/etc/nanorc").
			Symlink(fst.Tmp+"/etc/netgroup", "/etc/netgroup").
			Symlink(fst.Tmp+"/etc/NetworkManager", "/etc/NetworkManager").
			Symlink(fst.Tmp+"/etc/nix", "/etc/nix").
			Symlink(fst.Tmp+"/etc/nixos", "/etc/nixos").
			Symlink(fst.Tmp+"/etc/NIXOS", "/etc/NIXOS").
			Symlink(fst.Tmp+"/etc/nscd.conf", "/etc/nscd.conf").
			Symlink(fst.Tmp+"/etc/nsswitch.conf", "/etc/nsswitch.conf").
			Symlink(fst.Tmp+"/etc/opensnitchd", "/etc/opensnitchd").
			Symlink(fst.Tmp+"/etc/os-release", "/etc/os-release").
			Symlink(fst.Tmp+"/etc/pam", "/etc/pam").
			Symlink(fst.Tmp+"/etc/pam.d", "/etc/pam.d").
			Symlink(fst.Tmp+"/etc/pipewire", "/etc/pipewire").
			Symlink(fst.Tmp+"/etc/pki", "/etc/pki").
			Symlink(fst.Tmp+"/etc/polkit-1", "/etc/polkit-1").
			Symlink(fst.Tmp+"/etc/profile", "/etc/profile").
			Symlink(fst.Tmp+"/etc/protocols", "/etc/protocols").
			Symlink(fst.Tmp+"/etc/qemu", "/etc/qemu").
			Symlink(fst.Tmp+"/etc/resolv.conf", "/etc/resolv.conf").
			Symlink(fst.Tmp+"/etc/resolvconf.conf", "/etc/resolvconf.conf").
			Symlink(fst.Tmp+"/etc/rpc", "/etc/rpc").
			Symlink(fst.Tmp+"/etc/samba", "/etc/samba").
			Symlink(fst.Tmp+"/etc/sddm.conf", "/etc/sddm.conf").
			Symlink(fst.Tmp+"/etc/secureboot", "/etc/secureboot").
			Symlink(fst.Tmp+"/etc/services", "/etc/services").
			Symlink(fst.Tmp+"/etc/set-environment", "/etc/set-environment").
			Symlink(fst.Tmp+"/etc/shadow", "/etc/shadow").
			Symlink(fst.Tmp+"/etc/shells", "/etc/shells").
			Symlink(fst.Tmp+"/etc/ssh", "/etc/ssh").
			Symlink(fst.Tmp+"/etc/ssl", "/etc/ssl").
			Symlink(fst.Tmp+"/etc/static", "/etc/static").
			Symlink(fst.Tmp+"/etc/subgid", "/etc/subgid").
			Symlink(fst.Tmp+"/etc/subuid", "/etc/subuid").
			Symlink(fst.Tmp+"/etc/sudoers", "/etc/sudoers").
			Symlink(fst.Tmp+"/etc/sysctl.d", "/etc/sysctl.d").
			Symlink(fst.Tmp+"/etc/systemd", "/etc/systemd").
			Symlink(fst.Tmp+"/etc/terminfo", "/etc/terminfo").
			Symlink(fst.Tmp+"/etc/tmpfiles.d", "/etc/tmpfiles.d").
			Symlink(fst.Tmp+"/etc/udev", "/etc/udev").
			Symlink(fst.Tmp+"/etc/udisks2", "/etc/udisks2").
			Symlink(fst.Tmp+"/etc/UPower", "/etc/UPower").
			Symlink(fst.Tmp+"/etc/vconsole.conf", "/etc/vconsole.conf").
			Symlink(fst.Tmp+"/etc/X11", "/etc/X11").
			Symlink(fst.Tmp+"/etc/zfs", "/etc/zfs").
			Symlink(fst.Tmp+"/etc/zinputrc", "/etc/zinputrc").
			Symlink(fst.Tmp+"/etc/zoneinfo", "/etc/zoneinfo").
			Symlink(fst.Tmp+"/etc/zprofile", "/etc/zprofile").
			Symlink(fst.Tmp+"/etc/zshenv", "/etc/zshenv").
			Symlink(fst.Tmp+"/etc/zshrc", "/etc/zshrc").
			Bind("/tmp/fortify.1971/tmpdir/0", "/tmp", false, true).
			Tmpfs("/run/user", 1048576).
			Tmpfs("/run/user/65534", 8388608).
			Bind("/home/chronos", "/home/chronos", false, true).
			CopyBind("/etc/passwd", []byte("chronos:x:65534:65534:Fortify:/home/chronos:/run/current-system/sw/bin/zsh\n")).
			CopyBind("/etc/group", []byte("fortify:x:65534:\n")).
			Tmpfs("/var/run/nscd", 8192).
			Bind("/run/wrappers/bin/fortify", "/.fortify/sbin/fortify").
			Symlink("fortify", "/.fortify/sbin/init"),
	},
	{
		"nixos permissive defaults chromium", new(stubNixOS),
		&fst.Config{
			ID:      "org.chromium.Chromium",
			Command: []string{"/run/current-system/sw/bin/zsh", "-c", "exec chromium "},
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
			Ephemeral(system.Process, "/tmp/fortify.1971/ebf083d1b175911782d413369b64ce7c", 0711).
			Ensure("/tmp/fortify.1971/tmpdir", 0700).UpdatePermType(system.User, "/tmp/fortify.1971/tmpdir", acl.Execute).
			Ensure("/tmp/fortify.1971/tmpdir/9", 01700).UpdatePermType(system.User, "/tmp/fortify.1971/tmpdir/9", acl.Read, acl.Write, acl.Execute).
			Ensure("/run/user/1971/fortify", 0700).UpdatePermType(system.User, "/run/user/1971/fortify", acl.Execute).
			Ensure("/run/user/1971", 0700).UpdatePermType(system.User, "/run/user/1971", acl.Execute). // this is ordered as is because the previous Ensure only calls mkdir if XDG_RUNTIME_DIR is unset
			Ephemeral(system.Process, "/run/user/1971/fortify/ebf083d1b175911782d413369b64ce7c", 0700).UpdatePermType(system.Process, "/run/user/1971/fortify/ebf083d1b175911782d413369b64ce7c", acl.Execute).
			Ensure("/tmp/fortify.1971/wayland", 0711).
			Wayland("/tmp/fortify.1971/wayland/ebf083d1b175911782d413369b64ce7c", "/run/user/1971/wayland-0", "org.chromium.Chromium", "ebf083d1b175911782d413369b64ce7c").
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
		(&bwrap.Config{
			Net:      true,
			UserNS:   true,
			Chdir:    "/home/chronos",
			Clearenv: true,
			Syscall:  new(bwrap.SyscallPolicy),
			SetEnv: map[string]string{
				"DBUS_SESSION_BUS_ADDRESS": "unix:path=/run/user/65534/bus",
				"DBUS_SYSTEM_BUS_ADDRESS":  "unix:path=/run/dbus/system_bus_socket",
				"HOME":                     "/home/chronos",
				"PULSE_COOKIE":             fst.Tmp + "/pulse-cookie",
				"PULSE_SERVER":             "unix:/run/user/65534/pulse/native",
				"SHELL":                    "/run/current-system/sw/bin/zsh",
				"TERM":                     "xterm-256color",
				"USER":                     "chronos",
				"WAYLAND_DISPLAY":          "wayland-0",
				"XDG_RUNTIME_DIR":          "/run/user/65534",
				"XDG_SESSION_CLASS":        "user",
				"XDG_SESSION_TYPE":         "tty",
			},
			Chmod:         make(bwrap.ChmodConfig),
			DieWithParent: true,
			AsInit:        true,
		}).SetUID(65534).SetGID(65534).
			Procfs("/proc").
			Tmpfs(fst.Tmp, 4096).
			DevTmpfs("/dev").Mqueue("/dev/mqueue").
			Bind("/bin", "/bin", false, true).
			Bind("/boot", "/boot", false, true).
			Bind("/home", "/home", false, true).
			Bind("/lib", "/lib", false, true).
			Bind("/lib64", "/lib64", false, true).
			Bind("/nix", "/nix", false, true).
			Bind("/root", "/root", false, true).
			Bind("/run", "/run", false, true).
			Bind("/srv", "/srv", false, true).
			Bind("/sys", "/sys", false, true).
			Bind("/usr", "/usr", false, true).
			Bind("/var", "/var", false, true).
			Bind("/dev/dri", "/dev/dri", true, true, true).
			Bind("/dev/kvm", "/dev/kvm", true, true, true).
			Tmpfs("/run/user/1971", 8192).
			Tmpfs("/run/dbus", 8192).
			Bind("/etc", fst.Tmp+"/etc").
			Symlink(fst.Tmp+"/etc/alsa", "/etc/alsa").
			Symlink(fst.Tmp+"/etc/bashrc", "/etc/bashrc").
			Symlink(fst.Tmp+"/etc/binfmt.d", "/etc/binfmt.d").
			Symlink(fst.Tmp+"/etc/dbus-1", "/etc/dbus-1").
			Symlink(fst.Tmp+"/etc/default", "/etc/default").
			Symlink(fst.Tmp+"/etc/ethertypes", "/etc/ethertypes").
			Symlink(fst.Tmp+"/etc/fonts", "/etc/fonts").
			Symlink(fst.Tmp+"/etc/fstab", "/etc/fstab").
			Symlink(fst.Tmp+"/etc/fuse.conf", "/etc/fuse.conf").
			Symlink(fst.Tmp+"/etc/host.conf", "/etc/host.conf").
			Symlink(fst.Tmp+"/etc/hostid", "/etc/hostid").
			Symlink(fst.Tmp+"/etc/hostname", "/etc/hostname").
			Symlink(fst.Tmp+"/etc/hostname.CHECKSUM", "/etc/hostname.CHECKSUM").
			Symlink(fst.Tmp+"/etc/hosts", "/etc/hosts").
			Symlink(fst.Tmp+"/etc/inputrc", "/etc/inputrc").
			Symlink(fst.Tmp+"/etc/ipsec.d", "/etc/ipsec.d").
			Symlink(fst.Tmp+"/etc/issue", "/etc/issue").
			Symlink(fst.Tmp+"/etc/kbd", "/etc/kbd").
			Symlink(fst.Tmp+"/etc/libblockdev", "/etc/libblockdev").
			Symlink(fst.Tmp+"/etc/locale.conf", "/etc/locale.conf").
			Symlink(fst.Tmp+"/etc/localtime", "/etc/localtime").
			Symlink(fst.Tmp+"/etc/login.defs", "/etc/login.defs").
			Symlink(fst.Tmp+"/etc/lsb-release", "/etc/lsb-release").
			Symlink(fst.Tmp+"/etc/lvm", "/etc/lvm").
			Symlink(fst.Tmp+"/etc/machine-id", "/etc/machine-id").
			Symlink(fst.Tmp+"/etc/man_db.conf", "/etc/man_db.conf").
			Symlink(fst.Tmp+"/etc/modprobe.d", "/etc/modprobe.d").
			Symlink(fst.Tmp+"/etc/modules-load.d", "/etc/modules-load.d").
			Symlink("/proc/mounts", "/etc/mtab").
			Symlink(fst.Tmp+"/etc/nanorc", "/etc/nanorc").
			Symlink(fst.Tmp+"/etc/netgroup", "/etc/netgroup").
			Symlink(fst.Tmp+"/etc/NetworkManager", "/etc/NetworkManager").
			Symlink(fst.Tmp+"/etc/nix", "/etc/nix").
			Symlink(fst.Tmp+"/etc/nixos", "/etc/nixos").
			Symlink(fst.Tmp+"/etc/NIXOS", "/etc/NIXOS").
			Symlink(fst.Tmp+"/etc/nscd.conf", "/etc/nscd.conf").
			Symlink(fst.Tmp+"/etc/nsswitch.conf", "/etc/nsswitch.conf").
			Symlink(fst.Tmp+"/etc/opensnitchd", "/etc/opensnitchd").
			Symlink(fst.Tmp+"/etc/os-release", "/etc/os-release").
			Symlink(fst.Tmp+"/etc/pam", "/etc/pam").
			Symlink(fst.Tmp+"/etc/pam.d", "/etc/pam.d").
			Symlink(fst.Tmp+"/etc/pipewire", "/etc/pipewire").
			Symlink(fst.Tmp+"/etc/pki", "/etc/pki").
			Symlink(fst.Tmp+"/etc/polkit-1", "/etc/polkit-1").
			Symlink(fst.Tmp+"/etc/profile", "/etc/profile").
			Symlink(fst.Tmp+"/etc/protocols", "/etc/protocols").
			Symlink(fst.Tmp+"/etc/qemu", "/etc/qemu").
			Symlink(fst.Tmp+"/etc/resolv.conf", "/etc/resolv.conf").
			Symlink(fst.Tmp+"/etc/resolvconf.conf", "/etc/resolvconf.conf").
			Symlink(fst.Tmp+"/etc/rpc", "/etc/rpc").
			Symlink(fst.Tmp+"/etc/samba", "/etc/samba").
			Symlink(fst.Tmp+"/etc/sddm.conf", "/etc/sddm.conf").
			Symlink(fst.Tmp+"/etc/secureboot", "/etc/secureboot").
			Symlink(fst.Tmp+"/etc/services", "/etc/services").
			Symlink(fst.Tmp+"/etc/set-environment", "/etc/set-environment").
			Symlink(fst.Tmp+"/etc/shadow", "/etc/shadow").
			Symlink(fst.Tmp+"/etc/shells", "/etc/shells").
			Symlink(fst.Tmp+"/etc/ssh", "/etc/ssh").
			Symlink(fst.Tmp+"/etc/ssl", "/etc/ssl").
			Symlink(fst.Tmp+"/etc/static", "/etc/static").
			Symlink(fst.Tmp+"/etc/subgid", "/etc/subgid").
			Symlink(fst.Tmp+"/etc/subuid", "/etc/subuid").
			Symlink(fst.Tmp+"/etc/sudoers", "/etc/sudoers").
			Symlink(fst.Tmp+"/etc/sysctl.d", "/etc/sysctl.d").
			Symlink(fst.Tmp+"/etc/systemd", "/etc/systemd").
			Symlink(fst.Tmp+"/etc/terminfo", "/etc/terminfo").
			Symlink(fst.Tmp+"/etc/tmpfiles.d", "/etc/tmpfiles.d").
			Symlink(fst.Tmp+"/etc/udev", "/etc/udev").
			Symlink(fst.Tmp+"/etc/udisks2", "/etc/udisks2").
			Symlink(fst.Tmp+"/etc/UPower", "/etc/UPower").
			Symlink(fst.Tmp+"/etc/vconsole.conf", "/etc/vconsole.conf").
			Symlink(fst.Tmp+"/etc/X11", "/etc/X11").
			Symlink(fst.Tmp+"/etc/zfs", "/etc/zfs").
			Symlink(fst.Tmp+"/etc/zinputrc", "/etc/zinputrc").
			Symlink(fst.Tmp+"/etc/zoneinfo", "/etc/zoneinfo").
			Symlink(fst.Tmp+"/etc/zprofile", "/etc/zprofile").
			Symlink(fst.Tmp+"/etc/zshenv", "/etc/zshenv").
			Symlink(fst.Tmp+"/etc/zshrc", "/etc/zshrc").
			Bind("/tmp/fortify.1971/tmpdir/9", "/tmp", false, true).
			Tmpfs("/run/user", 1048576).
			Tmpfs("/run/user/65534", 8388608).
			Bind("/home/chronos", "/home/chronos", false, true).
			CopyBind("/etc/passwd", []byte("chronos:x:65534:65534:Fortify:/home/chronos:/run/current-system/sw/bin/zsh\n")).
			CopyBind("/etc/group", []byte("fortify:x:65534:\n")).
			Bind("/tmp/fortify.1971/wayland/ebf083d1b175911782d413369b64ce7c", "/run/user/65534/wayland-0").
			Bind("/run/user/1971/fortify/ebf083d1b175911782d413369b64ce7c/pulse", "/run/user/65534/pulse/native").
			CopyBind(fst.Tmp+"/pulse-cookie", nil).
			Bind("/tmp/fortify.1971/ebf083d1b175911782d413369b64ce7c/bus", "/run/user/65534/bus").
			Bind("/tmp/fortify.1971/ebf083d1b175911782d413369b64ce7c/system_bus_socket", "/run/dbus/system_bus_socket").
			Tmpfs("/var/run/nscd", 8192).
			Bind("/run/wrappers/bin/fortify", "/.fortify/sbin/fortify").
			Symlink("fortify", "/.fortify/sbin/init"),
	},
}
