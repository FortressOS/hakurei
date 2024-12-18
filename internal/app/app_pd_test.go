package app_test

import (
	"git.ophivana.moe/security/fortify/acl"
	"git.ophivana.moe/security/fortify/dbus"
	"git.ophivana.moe/security/fortify/fipc"
	"git.ophivana.moe/security/fortify/helper/bwrap"
	"git.ophivana.moe/security/fortify/internal/app"
	"git.ophivana.moe/security/fortify/internal/system"
)

var testCasesPd = []sealTestCase{
	{
		"nixos permissive defaults no enablements", new(stubNixOS),
		&fipc.Config{
			Command: make([]string, 0),
			Confinement: fipc.ConfinementConfig{
				AppID:    0,
				Username: "chronos",
				Outer:    "/home/chronos",
			},
		},
		app.ID{
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
			Ephemeral(system.Process, "/run/user/1971/fortify/4a450b6596d7bc15bd01780eb9a607ac", 0700).UpdatePermType(system.Process, "/run/user/1971/fortify/4a450b6596d7bc15bd01780eb9a607ac", acl.Execute).
			WriteType(system.Process, "/tmp/fortify.1971/4a450b6596d7bc15bd01780eb9a607ac/passwd", "chronos:x:65534:65534:Fortify:/home/chronos:/run/current-system/sw/bin/zsh\n").
			WriteType(system.Process, "/tmp/fortify.1971/4a450b6596d7bc15bd01780eb9a607ac/group", "fortify:x:65534:\n"),
		(&bwrap.Config{
			Net:      true,
			UserNS:   true,
			Clearenv: true,
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
			Tmpfs("/fortify", 4096).
			DevTmpfs("/dev").Mqueue("/dev/mqueue").
			Bind("/bin", "/bin", false, true).
			Bind("/boot", "/boot", false, true).
			Bind("/home", "/home", false, true).
			Bind("/lib", "/lib", false, true).
			Bind("/lib64", "/lib64", false, true).
			Bind("/nix", "/nix", false, true).
			Bind("/root", "/root", false, true).
			Bind("/srv", "/srv", false, true).
			Bind("/sys", "/sys", false, true).
			Bind("/usr", "/usr", false, true).
			Bind("/var", "/var", false, true).
			Bind("/run/agetty.reload", "/run/agetty.reload", false, true).
			Bind("/run/binfmt", "/run/binfmt", false, true).
			Bind("/run/booted-system", "/run/booted-system", false, true).
			Bind("/run/credentials", "/run/credentials", false, true).
			Bind("/run/cryptsetup", "/run/cryptsetup", false, true).
			Bind("/run/current-system", "/run/current-system", false, true).
			Bind("/run/host", "/run/host", false, true).
			Bind("/run/keys", "/run/keys", false, true).
			Bind("/run/libvirt", "/run/libvirt", false, true).
			Bind("/run/libvirtd.pid", "/run/libvirtd.pid", false, true).
			Bind("/run/lock", "/run/lock", false, true).
			Bind("/run/log", "/run/log", false, true).
			Bind("/run/lvm", "/run/lvm", false, true).
			Bind("/run/mount", "/run/mount", false, true).
			Bind("/run/NetworkManager", "/run/NetworkManager", false, true).
			Bind("/run/nginx", "/run/nginx", false, true).
			Bind("/run/nixos", "/run/nixos", false, true).
			Bind("/run/nscd", "/run/nscd", false, true).
			Bind("/run/opengl-driver", "/run/opengl-driver", false, true).
			Bind("/run/pppd", "/run/pppd", false, true).
			Bind("/run/resolvconf", "/run/resolvconf", false, true).
			Bind("/run/sddm", "/run/sddm", false, true).
			Bind("/run/store", "/run/store", false, true).
			Bind("/run/syncoid", "/run/syncoid", false, true).
			Bind("/run/system", "/run/system", false, true).
			Bind("/run/systemd", "/run/systemd", false, true).
			Bind("/run/tmpfiles.d", "/run/tmpfiles.d", false, true).
			Bind("/run/udev", "/run/udev", false, true).
			Bind("/run/udisks2", "/run/udisks2", false, true).
			Bind("/run/utmp", "/run/utmp", false, true).
			Bind("/run/virtlogd.pid", "/run/virtlogd.pid", false, true).
			Bind("/run/wrappers", "/run/wrappers", false, true).
			Bind("/run/zed.pid", "/run/zed.pid", false, true).
			Bind("/run/zed.state", "/run/zed.state", false, true).
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
			Bind("/tmp/fortify.1971/tmpdir/0", "/tmp", false, true).
			Tmpfs("/tmp/fortify.1971", 1048576).
			Tmpfs("/run/user", 1048576).
			Tmpfs("/run/user/65534", 8388608).
			Bind("/home/chronos", "/home/chronos", false, true).
			Bind("/tmp/fortify.1971/4a450b6596d7bc15bd01780eb9a607ac/passwd", "/etc/passwd").
			Bind("/tmp/fortify.1971/4a450b6596d7bc15bd01780eb9a607ac/group", "/etc/group").
			Tmpfs("/var/run/nscd", 8192),
	},
	{
		"nixos permissive defaults chromium", new(stubNixOS),
		&fipc.Config{
			ID:      "org.chromium.Chromium",
			Command: []string{"/run/current-system/sw/bin/zsh", "-c", "exec chromium "},
			Confinement: fipc.ConfinementConfig{
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
		app.ID{
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
			WriteType(system.Process, "/tmp/fortify.1971/ebf083d1b175911782d413369b64ce7c/passwd", "chronos:x:65534:65534:Fortify:/home/chronos:/run/current-system/sw/bin/zsh\n").
			WriteType(system.Process, "/tmp/fortify.1971/ebf083d1b175911782d413369b64ce7c/group", "fortify:x:65534:\n").
			Ensure("/tmp/fortify.1971/wayland", 0711).
			Wayland("/tmp/fortify.1971/wayland/ebf083d1b175911782d413369b64ce7c", "/run/user/1971/wayland-0", "org.chromium.Chromium", "ebf083d1b175911782d413369b64ce7c").
			Link("/run/user/1971/pulse/native", "/run/user/1971/fortify/ebf083d1b175911782d413369b64ce7c/pulse").
			CopyFile("/tmp/fortify.1971/ebf083d1b175911782d413369b64ce7c/pulse-cookie", "/home/ophestra/xdg/config/pulse/cookie").
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
			SetEnv: map[string]string{
				"DBUS_SESSION_BUS_ADDRESS": "unix:path=/run/user/65534/bus",
				"DBUS_SYSTEM_BUS_ADDRESS":  "unix:path=/run/dbus/system_bus_socket",
				"HOME":                     "/home/chronos",
				"PULSE_COOKIE":             "/tmp/fortify.1971/ebf083d1b175911782d413369b64ce7c/pulse-cookie",
				"PULSE_SERVER":             "unix:/run/user/65534/pulse/native",
				"SHELL":                    "/run/current-system/sw/bin/zsh",
				"TERM":                     "xterm-256color",
				"USER":                     "chronos",
				"WAYLAND_DISPLAY":          "/run/user/65534/wayland-0",
				"XDG_RUNTIME_DIR":          "/run/user/65534",
				"XDG_SESSION_CLASS":        "user",
				"XDG_SESSION_TYPE":         "tty",
			},
			Chmod:         make(bwrap.ChmodConfig),
			DieWithParent: true,
			AsInit:        true,
		}).SetUID(65534).SetGID(65534).
			Procfs("/proc").
			Tmpfs("/fortify", 4096).
			DevTmpfs("/dev").Mqueue("/dev/mqueue").
			Bind("/bin", "/bin", false, true).
			Bind("/boot", "/boot", false, true).
			Bind("/home", "/home", false, true).
			Bind("/lib", "/lib", false, true).
			Bind("/lib64", "/lib64", false, true).
			Bind("/nix", "/nix", false, true).
			Bind("/root", "/root", false, true).
			Bind("/srv", "/srv", false, true).
			Bind("/sys", "/sys", false, true).
			Bind("/usr", "/usr", false, true).
			Bind("/var", "/var", false, true).
			Bind("/run/agetty.reload", "/run/agetty.reload", false, true).
			Bind("/run/binfmt", "/run/binfmt", false, true).
			Bind("/run/booted-system", "/run/booted-system", false, true).
			Bind("/run/credentials", "/run/credentials", false, true).
			Bind("/run/cryptsetup", "/run/cryptsetup", false, true).
			Bind("/run/current-system", "/run/current-system", false, true).
			Bind("/run/host", "/run/host", false, true).
			Bind("/run/keys", "/run/keys", false, true).
			Bind("/run/libvirt", "/run/libvirt", false, true).
			Bind("/run/libvirtd.pid", "/run/libvirtd.pid", false, true).
			Bind("/run/lock", "/run/lock", false, true).
			Bind("/run/log", "/run/log", false, true).
			Bind("/run/lvm", "/run/lvm", false, true).
			Bind("/run/mount", "/run/mount", false, true).
			Bind("/run/NetworkManager", "/run/NetworkManager", false, true).
			Bind("/run/nginx", "/run/nginx", false, true).
			Bind("/run/nixos", "/run/nixos", false, true).
			Bind("/run/nscd", "/run/nscd", false, true).
			Bind("/run/opengl-driver", "/run/opengl-driver", false, true).
			Bind("/run/pppd", "/run/pppd", false, true).
			Bind("/run/resolvconf", "/run/resolvconf", false, true).
			Bind("/run/sddm", "/run/sddm", false, true).
			Bind("/run/store", "/run/store", false, true).
			Bind("/run/syncoid", "/run/syncoid", false, true).
			Bind("/run/system", "/run/system", false, true).
			Bind("/run/systemd", "/run/systemd", false, true).
			Bind("/run/tmpfiles.d", "/run/tmpfiles.d", false, true).
			Bind("/run/udev", "/run/udev", false, true).
			Bind("/run/udisks2", "/run/udisks2", false, true).
			Bind("/run/utmp", "/run/utmp", false, true).
			Bind("/run/virtlogd.pid", "/run/virtlogd.pid", false, true).
			Bind("/run/wrappers", "/run/wrappers", false, true).
			Bind("/run/zed.pid", "/run/zed.pid", false, true).
			Bind("/run/zed.state", "/run/zed.state", false, true).
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
			Bind("/tmp/fortify.1971/tmpdir/9", "/tmp", false, true).
			Tmpfs("/tmp/fortify.1971", 1048576).
			Tmpfs("/run/user", 1048576).
			Tmpfs("/run/user/65534", 8388608).
			Bind("/home/chronos", "/home/chronos", false, true).
			Bind("/tmp/fortify.1971/ebf083d1b175911782d413369b64ce7c/passwd", "/etc/passwd").
			Bind("/tmp/fortify.1971/ebf083d1b175911782d413369b64ce7c/group", "/etc/group").
			Bind("/tmp/fortify.1971/wayland/ebf083d1b175911782d413369b64ce7c", "/run/user/65534/wayland-0").
			Bind("/run/user/1971/fortify/ebf083d1b175911782d413369b64ce7c/pulse", "/run/user/65534/pulse/native").
			Bind("/tmp/fortify.1971/ebf083d1b175911782d413369b64ce7c/pulse-cookie", "/tmp/fortify.1971/ebf083d1b175911782d413369b64ce7c/pulse-cookie").
			Bind("/tmp/fortify.1971/ebf083d1b175911782d413369b64ce7c/bus", "/run/user/65534/bus").
			Bind("/tmp/fortify.1971/ebf083d1b175911782d413369b64ce7c/system_bus_socket", "/run/dbus/system_bus_socket").
			Tmpfs("/var/run/nscd", 8192),
	},
}
