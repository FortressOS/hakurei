package container

import (
	"errors"
	"os"
	"testing"

	"hakurei.app/container/check"
	"hakurei.app/container/stub"
)

func TestAutoEtcOp(t *testing.T) {
	t.Run("nonrepeatable", func(t *testing.T) {
		wantErr := OpRepeatError("autoetc")
		if err := (&AutoEtcOp{Prefix: "81ceabb30d37bbdb3868004629cb84e9"}).apply(&setupState{nonrepeatable: nrAutoEtc}, nil); !errors.Is(err, wantErr) {
			t.Errorf("apply: error = %v, want %v", err, wantErr)
		}
	})

	checkOpBehaviour(t, []opBehaviourTestCase{
		{"mkdirAll", new(Params), &AutoEtcOp{
			Prefix: "81ceabb30d37bbdb3868004629cb84e9",
		}, nil, nil, []stub.Call{
			call("mkdirAll", stub.ExpectArgs{"/sysroot/etc/", os.FileMode(0755)}, nil, stub.UniqueError(3)),
		}, stub.UniqueError(3)},

		{"readdir", new(Params), &AutoEtcOp{
			Prefix: "81ceabb30d37bbdb3868004629cb84e9",
		}, nil, nil, []stub.Call{
			call("mkdirAll", stub.ExpectArgs{"/sysroot/etc/", os.FileMode(0755)}, nil, nil),
			call("readdir", stub.ExpectArgs{"/sysroot/etc/.host/81ceabb30d37bbdb3868004629cb84e9"}, stubDir(), stub.UniqueError(2)),
		}, stub.UniqueError(2)},

		{"symlink", new(Params), &AutoEtcOp{
			Prefix: "81ceabb30d37bbdb3868004629cb84e9",
		}, nil, nil, []stub.Call{
			call("mkdirAll", stub.ExpectArgs{"/sysroot/etc/", os.FileMode(0755)}, nil, nil),
			call("readdir", stub.ExpectArgs{"/sysroot/etc/.host/81ceabb30d37bbdb3868004629cb84e9"}, stubDir(".host",
				"alsa", "bash_logout", "bashrc", "binfmt.d", "dbus-1", "default", "dhcpcd.exit-hook", "fonts",
				"fstab", "fuse.conf", "group", "host.conf", "hostname", "hosts", "hsurc", "inputrc", "issue", "kbd",
				"locale.conf", "login.defs", "lsb-release", "lvm", "machine-id", "man_db.conf", "mdadm.conf",
				"modprobe.d", "modules-load.d", "mtab", "nanorc", "netgroup", "nix", "nixos", "NIXOS", "nscd.conf",
				"nsswitch.conf", "os-release", "pam", "pam.d", "passwd", "pipewire", "pki", "polkit-1", "profile",
				"protocols", "resolv.conf", "resolvconf.conf", "rpc", "services", "set-environment", "shadow", "shells",
				"ssh", "ssl", "static", "subgid", "subuid", "sudoers", "sway", "sysctl.d", "systemd", "terminfo",
				"tmpfiles.d", "udev", "vconsole.conf", "X11", "xdg", "zoneinfo"), nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/alsa", "/sysroot/etc/alsa"}, nil, stub.UniqueError(1)),
		}, stub.UniqueError(1)},

		{"symlink mtab", new(Params), &AutoEtcOp{
			Prefix: "81ceabb30d37bbdb3868004629cb84e9",
		}, nil, nil, []stub.Call{
			call("mkdirAll", stub.ExpectArgs{"/sysroot/etc/", os.FileMode(0755)}, nil, nil),
			call("readdir", stub.ExpectArgs{"/sysroot/etc/.host/81ceabb30d37bbdb3868004629cb84e9"}, stubDir(".host",
				"alsa", "bash_logout", "bashrc", "binfmt.d", "dbus-1", "default", "dhcpcd.exit-hook", "fonts",
				"fstab", "fuse.conf", "group", "host.conf", "hostname", "hosts", "hsurc", "inputrc", "issue", "kbd",
				"locale.conf", "login.defs", "lsb-release", "lvm", "machine-id", "man_db.conf", "mdadm.conf",
				"modprobe.d", "modules-load.d", "mtab", "nanorc", "netgroup", "nix", "nixos", "NIXOS", "nscd.conf",
				"nsswitch.conf", "os-release", "pam", "pam.d", "passwd", "pipewire", "pki", "polkit-1", "profile",
				"protocols", "resolv.conf", "resolvconf.conf", "rpc", "services", "set-environment", "shadow", "shells",
				"ssh", "ssl", "static", "subgid", "subuid", "sudoers", "sway", "sysctl.d", "systemd", "terminfo",
				"tmpfiles.d", "udev", "vconsole.conf", "X11", "xdg", "zoneinfo"), nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/alsa", "/sysroot/etc/alsa"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/bash_logout", "/sysroot/etc/bash_logout"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/bashrc", "/sysroot/etc/bashrc"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/binfmt.d", "/sysroot/etc/binfmt.d"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/dbus-1", "/sysroot/etc/dbus-1"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/default", "/sysroot/etc/default"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/dhcpcd.exit-hook", "/sysroot/etc/dhcpcd.exit-hook"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/fonts", "/sysroot/etc/fonts"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/fstab", "/sysroot/etc/fstab"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/fuse.conf", "/sysroot/etc/fuse.conf"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/host.conf", "/sysroot/etc/host.conf"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/hostname", "/sysroot/etc/hostname"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/hosts", "/sysroot/etc/hosts"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/hsurc", "/sysroot/etc/hsurc"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/inputrc", "/sysroot/etc/inputrc"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/issue", "/sysroot/etc/issue"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/kbd", "/sysroot/etc/kbd"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/locale.conf", "/sysroot/etc/locale.conf"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/login.defs", "/sysroot/etc/login.defs"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/lsb-release", "/sysroot/etc/lsb-release"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/lvm", "/sysroot/etc/lvm"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/machine-id", "/sysroot/etc/machine-id"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/man_db.conf", "/sysroot/etc/man_db.conf"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/mdadm.conf", "/sysroot/etc/mdadm.conf"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/modprobe.d", "/sysroot/etc/modprobe.d"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/modules-load.d", "/sysroot/etc/modules-load.d"}, nil, nil),
			call("symlink", stub.ExpectArgs{"/proc/mounts", "/sysroot/etc/mtab"}, nil, stub.UniqueError(0)),
		}, stub.UniqueError(0)},

		{"success nested", new(Params), &AutoEtcOp{
			Prefix: "81ceabb30d37bbdb3868004629cb84e9",
		}, nil, nil, []stub.Call{
			call("mkdirAll", stub.ExpectArgs{"/sysroot/etc/", os.FileMode(0755)}, nil, nil),
			call("readdir", stub.ExpectArgs{"/sysroot/etc/.host/81ceabb30d37bbdb3868004629cb84e9"}, stubDir(".host",
				"alsa", "bash_logout", "bashrc", "binfmt.d", "dbus-1", "default", "dhcpcd.exit-hook", "fonts",
				"fstab", "fuse.conf", "group", "host.conf", "hostname", "hosts", "hsurc", "inputrc", "issue", "kbd",
				"locale.conf", "login.defs", "lsb-release", "lvm", "machine-id", "man_db.conf", "mdadm.conf",
				"modprobe.d", "modules-load.d", "mtab", "nanorc", "netgroup", "nix", "nixos", "NIXOS", "nscd.conf",
				"nsswitch.conf", "os-release", "pam", "pam.d", "passwd", "pipewire", "pki", "polkit-1", "profile",
				"protocols", "resolv.conf", "resolvconf.conf", "rpc", "services", "set-environment", "shadow", "shells",
				"ssh", "ssl", "static", "subgid", "subuid", "sudoers", "sway", "sysctl.d", "systemd", "terminfo",
				"tmpfiles.d", "udev", "vconsole.conf", "X11", "xdg", "zoneinfo"), nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/alsa", "/sysroot/etc/alsa"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/bash_logout", "/sysroot/etc/bash_logout"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/bashrc", "/sysroot/etc/bashrc"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/binfmt.d", "/sysroot/etc/binfmt.d"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/dbus-1", "/sysroot/etc/dbus-1"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/default", "/sysroot/etc/default"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/dhcpcd.exit-hook", "/sysroot/etc/dhcpcd.exit-hook"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/fonts", "/sysroot/etc/fonts"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/fstab", "/sysroot/etc/fstab"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/fuse.conf", "/sysroot/etc/fuse.conf"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/host.conf", "/sysroot/etc/host.conf"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/hostname", "/sysroot/etc/hostname"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/hosts", "/sysroot/etc/hosts"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/hsurc", "/sysroot/etc/hsurc"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/inputrc", "/sysroot/etc/inputrc"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/issue", "/sysroot/etc/issue"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/kbd", "/sysroot/etc/kbd"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/locale.conf", "/sysroot/etc/locale.conf"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/login.defs", "/sysroot/etc/login.defs"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/lsb-release", "/sysroot/etc/lsb-release"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/lvm", "/sysroot/etc/lvm"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/machine-id", "/sysroot/etc/machine-id"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/man_db.conf", "/sysroot/etc/man_db.conf"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/mdadm.conf", "/sysroot/etc/mdadm.conf"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/modprobe.d", "/sysroot/etc/modprobe.d"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/modules-load.d", "/sysroot/etc/modules-load.d"}, nil, nil),
			call("symlink", stub.ExpectArgs{"/proc/mounts", "/sysroot/etc/mtab"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/nanorc", "/sysroot/etc/nanorc"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/netgroup", "/sysroot/etc/netgroup"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/nix", "/sysroot/etc/nix"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/nixos", "/sysroot/etc/nixos"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/NIXOS", "/sysroot/etc/NIXOS"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/nscd.conf", "/sysroot/etc/nscd.conf"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/nsswitch.conf", "/sysroot/etc/nsswitch.conf"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/os-release", "/sysroot/etc/os-release"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/pam", "/sysroot/etc/pam"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/pam.d", "/sysroot/etc/pam.d"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/pipewire", "/sysroot/etc/pipewire"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/pki", "/sysroot/etc/pki"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/polkit-1", "/sysroot/etc/polkit-1"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/profile", "/sysroot/etc/profile"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/protocols", "/sysroot/etc/protocols"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/resolv.conf", "/sysroot/etc/resolv.conf"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/resolvconf.conf", "/sysroot/etc/resolvconf.conf"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/rpc", "/sysroot/etc/rpc"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/services", "/sysroot/etc/services"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/set-environment", "/sysroot/etc/set-environment"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/shadow", "/sysroot/etc/shadow"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/shells", "/sysroot/etc/shells"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/ssh", "/sysroot/etc/ssh"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/ssl", "/sysroot/etc/ssl"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/static", "/sysroot/etc/static"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/subgid", "/sysroot/etc/subgid"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/subuid", "/sysroot/etc/subuid"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/sudoers", "/sysroot/etc/sudoers"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/sway", "/sysroot/etc/sway"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/sysctl.d", "/sysroot/etc/sysctl.d"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/systemd", "/sysroot/etc/systemd"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/terminfo", "/sysroot/etc/terminfo"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/tmpfiles.d", "/sysroot/etc/tmpfiles.d"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/udev", "/sysroot/etc/udev"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/vconsole.conf", "/sysroot/etc/vconsole.conf"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/X11", "/sysroot/etc/X11"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/xdg", "/sysroot/etc/xdg"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/zoneinfo", "/sysroot/etc/zoneinfo"}, nil, nil),
		}, nil},

		{"success", new(Params), &AutoEtcOp{
			Prefix: "81ceabb30d37bbdb3868004629cb84e9",
		}, nil, nil, []stub.Call{
			call("mkdirAll", stub.ExpectArgs{"/sysroot/etc/", os.FileMode(0755)}, nil, nil),
			call("readdir", stub.ExpectArgs{"/sysroot/etc/.host/81ceabb30d37bbdb3868004629cb84e9"}, stubDir(
				"alsa", "bash_logout", "bashrc", "binfmt.d", "dbus-1", "default", "dhcpcd.exit-hook", "fonts",
				"fstab", "fuse.conf", "group", "host.conf", "hostname", "hosts", "hsurc", "inputrc", "issue", "kbd",
				"locale.conf", "login.defs", "lsb-release", "lvm", "machine-id", "man_db.conf", "mdadm.conf",
				"modprobe.d", "modules-load.d", "mtab", "nanorc", "netgroup", "nix", "nixos", "NIXOS", "nscd.conf",
				"nsswitch.conf", "os-release", "pam", "pam.d", "passwd", "pipewire", "pki", "polkit-1", "profile",
				"protocols", "resolv.conf", "resolvconf.conf", "rpc", "services", "set-environment", "shadow", "shells",
				"ssh", "ssl", "static", "subgid", "subuid", "sudoers", "sway", "sysctl.d", "systemd", "terminfo",
				"tmpfiles.d", "udev", "vconsole.conf", "X11", "xdg", "zoneinfo"), nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/alsa", "/sysroot/etc/alsa"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/bash_logout", "/sysroot/etc/bash_logout"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/bashrc", "/sysroot/etc/bashrc"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/binfmt.d", "/sysroot/etc/binfmt.d"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/dbus-1", "/sysroot/etc/dbus-1"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/default", "/sysroot/etc/default"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/dhcpcd.exit-hook", "/sysroot/etc/dhcpcd.exit-hook"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/fonts", "/sysroot/etc/fonts"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/fstab", "/sysroot/etc/fstab"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/fuse.conf", "/sysroot/etc/fuse.conf"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/host.conf", "/sysroot/etc/host.conf"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/hostname", "/sysroot/etc/hostname"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/hosts", "/sysroot/etc/hosts"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/hsurc", "/sysroot/etc/hsurc"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/inputrc", "/sysroot/etc/inputrc"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/issue", "/sysroot/etc/issue"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/kbd", "/sysroot/etc/kbd"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/locale.conf", "/sysroot/etc/locale.conf"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/login.defs", "/sysroot/etc/login.defs"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/lsb-release", "/sysroot/etc/lsb-release"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/lvm", "/sysroot/etc/lvm"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/machine-id", "/sysroot/etc/machine-id"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/man_db.conf", "/sysroot/etc/man_db.conf"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/mdadm.conf", "/sysroot/etc/mdadm.conf"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/modprobe.d", "/sysroot/etc/modprobe.d"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/modules-load.d", "/sysroot/etc/modules-load.d"}, nil, nil),
			call("symlink", stub.ExpectArgs{"/proc/mounts", "/sysroot/etc/mtab"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/nanorc", "/sysroot/etc/nanorc"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/netgroup", "/sysroot/etc/netgroup"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/nix", "/sysroot/etc/nix"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/nixos", "/sysroot/etc/nixos"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/NIXOS", "/sysroot/etc/NIXOS"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/nscd.conf", "/sysroot/etc/nscd.conf"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/nsswitch.conf", "/sysroot/etc/nsswitch.conf"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/os-release", "/sysroot/etc/os-release"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/pam", "/sysroot/etc/pam"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/pam.d", "/sysroot/etc/pam.d"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/pipewire", "/sysroot/etc/pipewire"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/pki", "/sysroot/etc/pki"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/polkit-1", "/sysroot/etc/polkit-1"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/profile", "/sysroot/etc/profile"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/protocols", "/sysroot/etc/protocols"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/resolv.conf", "/sysroot/etc/resolv.conf"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/resolvconf.conf", "/sysroot/etc/resolvconf.conf"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/rpc", "/sysroot/etc/rpc"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/services", "/sysroot/etc/services"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/set-environment", "/sysroot/etc/set-environment"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/shadow", "/sysroot/etc/shadow"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/shells", "/sysroot/etc/shells"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/ssh", "/sysroot/etc/ssh"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/ssl", "/sysroot/etc/ssl"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/static", "/sysroot/etc/static"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/subgid", "/sysroot/etc/subgid"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/subuid", "/sysroot/etc/subuid"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/sudoers", "/sysroot/etc/sudoers"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/sway", "/sysroot/etc/sway"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/sysctl.d", "/sysroot/etc/sysctl.d"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/systemd", "/sysroot/etc/systemd"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/terminfo", "/sysroot/etc/terminfo"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/tmpfiles.d", "/sysroot/etc/tmpfiles.d"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/udev", "/sysroot/etc/udev"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/vconsole.conf", "/sysroot/etc/vconsole.conf"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/X11", "/sysroot/etc/X11"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/xdg", "/sysroot/etc/xdg"}, nil, nil),
			call("symlink", stub.ExpectArgs{".host/81ceabb30d37bbdb3868004629cb84e9/zoneinfo", "/sysroot/etc/zoneinfo"}, nil, nil),
		}, nil},
	})

	checkOpsValid(t, []opValidTestCase{
		{"nil", (*AutoEtcOp)(nil), false},
		{"zero", new(AutoEtcOp), true},
		{"populated", &AutoEtcOp{Prefix: ":3"}, true},
	})

	checkOpsBuilder(t, []opsBuilderTestCase{
		{"pd", new(Ops).Etc(check.MustAbs("/etc/"), "048090b6ed8f9ebb10e275ff5d8c0659"), Ops{
			&MkdirOp{Path: check.MustAbs("/etc/"), Perm: 0755},
			&BindMountOp{
				Source: check.MustAbs("/etc/"),
				Target: check.MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
			},
			&AutoEtcOp{Prefix: "048090b6ed8f9ebb10e275ff5d8c0659"},
		}},
	})

	checkOpIs(t, []opIsTestCase{
		{"zero", new(AutoEtcOp), new(AutoEtcOp), true},
		{"differs", &AutoEtcOp{Prefix: "\x00"}, &AutoEtcOp{":3"}, false},
		{"equals", &AutoEtcOp{Prefix: ":3"}, &AutoEtcOp{":3"}, true},
	})

	checkOpMeta(t, []opMetaTestCase{
		{"etc", &AutoEtcOp{
			Prefix: ":3",
		}, "setting up", "auto etc :3"},
	})

	t.Run("host path rel", func(t *testing.T) {
		op := &AutoEtcOp{Prefix: "048090b6ed8f9ebb10e275ff5d8c0659"}
		wantHostPath := "/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"
		wantHostRel := ".host/048090b6ed8f9ebb10e275ff5d8c0659"
		if got := op.hostPath(); got.String() != wantHostPath {
			t.Errorf("hostPath: %q, want %q", got, wantHostPath)
		}
		if got := op.hostRel(); got != wantHostRel {
			t.Errorf("hostRel: %q, want %q", got, wantHostRel)
		}
	})
}
