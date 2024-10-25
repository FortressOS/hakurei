package app_test

import (
	"fmt"
	"io/fs"
	"os/user"
	"strconv"

	"git.ophivana.moe/security/fortify/acl"
	"git.ophivana.moe/security/fortify/helper/bwrap"
	"git.ophivana.moe/security/fortify/internal"
	"git.ophivana.moe/security/fortify/internal/app"
	"git.ophivana.moe/security/fortify/internal/system"
)

var testCasesNixos = []sealTestCase{
	{
		"nixos permissive defaults no enablements", new(stubNixOS),
		&app.Config{
			User:    "chronos",
			Command: make([]string, 0),
			Method:  "sudo",
		},
		app.ID{
			0x4a, 0x45, 0x0b, 0x65,
			0x96, 0xd7, 0xbc, 0x15,
			0xbd, 0x01, 0x78, 0x0e,
			0xb9, 0xa6, 0x07, 0xac,
		},
		system.New(150).
			Ensure("/tmp/fortify.1971", 0701).
			Ephemeral(system.Process, "/tmp/fortify.1971/4a450b6596d7bc15bd01780eb9a607ac", 0701).
			Ensure("/tmp/fortify.1971/tmpdir", 0700).UpdatePermType(system.User, "/tmp/fortify.1971/tmpdir", acl.Execute).
			Ensure("/tmp/fortify.1971/tmpdir/150", 01700).UpdatePermType(system.User, "/tmp/fortify.1971/tmpdir/150", acl.Read, acl.Write, acl.Execute).
			Ensure("/run/user/1971/fortify", 0700).UpdatePermType(system.User, "/run/user/1971/fortify", acl.Execute).
			Ensure("/run/user/1971", 0700).UpdatePermType(system.User, "/run/user/1971", acl.Execute). // this is ordered as is because the previous Ensure only calls mkdir if XDG_RUNTIME_DIR is unset
			Ephemeral(system.Process, "/run/user/1971/fortify/4a450b6596d7bc15bd01780eb9a607ac", 0700).UpdatePermType(system.Process, "/run/user/1971/fortify/4a450b6596d7bc15bd01780eb9a607ac", acl.Execute).
			WriteType(system.Process, "/tmp/fortify.1971/4a450b6596d7bc15bd01780eb9a607ac/passwd", "chronos:x:65534:65534:Fortify:/home/chronos:/run/current-system/sw/bin/zsh\n").
			WriteType(system.Process, "/tmp/fortify.1971/4a450b6596d7bc15bd01780eb9a607ac/group", "fortify:x:65534:\n"),
		(&bwrap.Config{
			Net:      true,
			UserNS:   true,
			Clearenv: true,
			SetEnv: map[string]string{
				"HOME":              "/home/chronos",
				"SHELL":             "/run/current-system/sw/bin/zsh",
				"TERM":              "xterm-256color",
				"USER":              "chronos",
				"XDG_RUNTIME_DIR":   "/run/user/150",
				"XDG_SESSION_CLASS": "user",
				"XDG_SESSION_TYPE":  "tty"},
			Chmod:         make(bwrap.ChmodConfig),
			DieWithParent: true,
			AsInit:        true,
		}).SetUID(65534).SetGID(65534).
			Procfs("/proc").DevTmpfs("/dev").Mqueue("/dev/mqueue").
			Tmpfs("/dev/fortify", 4096).
			Bind("/bin", "/bin", false, true).
			Bind("/boot", "/boot", false, true).
			Bind("/etc", "/dev/fortify/etc").
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
			Symlink("/dev/fortify/etc/alsa", "/etc/alsa").
			Symlink("/dev/fortify/etc/bashrc", "/etc/bashrc").
			Symlink("/dev/fortify/etc/binfmt.d", "/etc/binfmt.d").
			Symlink("/dev/fortify/etc/dbus-1", "/etc/dbus-1").
			Symlink("/dev/fortify/etc/default", "/etc/default").
			Symlink("/dev/fortify/etc/ethertypes", "/etc/ethertypes").
			Symlink("/dev/fortify/etc/fonts", "/etc/fonts").
			Symlink("/dev/fortify/etc/fstab", "/etc/fstab").
			Symlink("/dev/fortify/etc/fuse.conf", "/etc/fuse.conf").
			Symlink("/dev/fortify/etc/host.conf", "/etc/host.conf").
			Symlink("/dev/fortify/etc/hostid", "/etc/hostid").
			Symlink("/dev/fortify/etc/hostname", "/etc/hostname").
			Symlink("/dev/fortify/etc/hostname.CHECKSUM", "/etc/hostname.CHECKSUM").
			Symlink("/dev/fortify/etc/hosts", "/etc/hosts").
			Symlink("/dev/fortify/etc/inputrc", "/etc/inputrc").
			Symlink("/dev/fortify/etc/ipsec.d", "/etc/ipsec.d").
			Symlink("/dev/fortify/etc/issue", "/etc/issue").
			Symlink("/dev/fortify/etc/kbd", "/etc/kbd").
			Symlink("/dev/fortify/etc/libblockdev", "/etc/libblockdev").
			Symlink("/dev/fortify/etc/locale.conf", "/etc/locale.conf").
			Symlink("/dev/fortify/etc/localtime", "/etc/localtime").
			Symlink("/dev/fortify/etc/login.defs", "/etc/login.defs").
			Symlink("/dev/fortify/etc/lsb-release", "/etc/lsb-release").
			Symlink("/dev/fortify/etc/lvm", "/etc/lvm").
			Symlink("/dev/fortify/etc/machine-id", "/etc/machine-id").
			Symlink("/dev/fortify/etc/man_db.conf", "/etc/man_db.conf").
			Symlink("/dev/fortify/etc/modprobe.d", "/etc/modprobe.d").
			Symlink("/dev/fortify/etc/modules-load.d", "/etc/modules-load.d").
			Symlink("/proc/mounts", "/etc/mtab").
			Symlink("/dev/fortify/etc/nanorc", "/etc/nanorc").
			Symlink("/dev/fortify/etc/netgroup", "/etc/netgroup").
			Symlink("/dev/fortify/etc/NetworkManager", "/etc/NetworkManager").
			Symlink("/dev/fortify/etc/nix", "/etc/nix").
			Symlink("/dev/fortify/etc/nixos", "/etc/nixos").
			Symlink("/dev/fortify/etc/NIXOS", "/etc/NIXOS").
			Symlink("/dev/fortify/etc/nscd.conf", "/etc/nscd.conf").
			Symlink("/dev/fortify/etc/nsswitch.conf", "/etc/nsswitch.conf").
			Symlink("/dev/fortify/etc/opensnitchd", "/etc/opensnitchd").
			Symlink("/dev/fortify/etc/os-release", "/etc/os-release").
			Symlink("/dev/fortify/etc/pam", "/etc/pam").
			Symlink("/dev/fortify/etc/pam.d", "/etc/pam.d").
			Symlink("/dev/fortify/etc/pipewire", "/etc/pipewire").
			Symlink("/dev/fortify/etc/pki", "/etc/pki").
			Symlink("/dev/fortify/etc/polkit-1", "/etc/polkit-1").
			Symlink("/dev/fortify/etc/profile", "/etc/profile").
			Symlink("/dev/fortify/etc/protocols", "/etc/protocols").
			Symlink("/dev/fortify/etc/qemu", "/etc/qemu").
			Symlink("/dev/fortify/etc/resolv.conf", "/etc/resolv.conf").
			Symlink("/dev/fortify/etc/resolvconf.conf", "/etc/resolvconf.conf").
			Symlink("/dev/fortify/etc/rpc", "/etc/rpc").
			Symlink("/dev/fortify/etc/samba", "/etc/samba").
			Symlink("/dev/fortify/etc/sddm.conf", "/etc/sddm.conf").
			Symlink("/dev/fortify/etc/secureboot", "/etc/secureboot").
			Symlink("/dev/fortify/etc/services", "/etc/services").
			Symlink("/dev/fortify/etc/set-environment", "/etc/set-environment").
			Symlink("/dev/fortify/etc/shadow", "/etc/shadow").
			Symlink("/dev/fortify/etc/shells", "/etc/shells").
			Symlink("/dev/fortify/etc/ssh", "/etc/ssh").
			Symlink("/dev/fortify/etc/ssl", "/etc/ssl").
			Symlink("/dev/fortify/etc/static", "/etc/static").
			Symlink("/dev/fortify/etc/subgid", "/etc/subgid").
			Symlink("/dev/fortify/etc/subuid", "/etc/subuid").
			Symlink("/dev/fortify/etc/sudoers", "/etc/sudoers").
			Symlink("/dev/fortify/etc/sysctl.d", "/etc/sysctl.d").
			Symlink("/dev/fortify/etc/systemd", "/etc/systemd").
			Symlink("/dev/fortify/etc/terminfo", "/etc/terminfo").
			Symlink("/dev/fortify/etc/tmpfiles.d", "/etc/tmpfiles.d").
			Symlink("/dev/fortify/etc/udev", "/etc/udev").
			Symlink("/dev/fortify/etc/udisks2", "/etc/udisks2").
			Symlink("/dev/fortify/etc/UPower", "/etc/UPower").
			Symlink("/dev/fortify/etc/vconsole.conf", "/etc/vconsole.conf").
			Symlink("/dev/fortify/etc/X11", "/etc/X11").
			Symlink("/dev/fortify/etc/zfs", "/etc/zfs").
			Symlink("/dev/fortify/etc/zinputrc", "/etc/zinputrc").
			Symlink("/dev/fortify/etc/zoneinfo", "/etc/zoneinfo").
			Symlink("/dev/fortify/etc/zprofile", "/etc/zprofile").
			Symlink("/dev/fortify/etc/zshenv", "/etc/zshenv").
			Symlink("/dev/fortify/etc/zshrc", "/etc/zshrc").
			Bind("/tmp/fortify.1971/tmpdir/150", "/tmp", false, true).
			Tmpfs("/tmp/fortify.1971", 1048576).
			Tmpfs("/run/user", 1048576).
			Tmpfs("/run/user/150", 8388608).
			Bind("/tmp/fortify.1971/4a450b6596d7bc15bd01780eb9a607ac/passwd", "/etc/passwd").
			Bind("/tmp/fortify.1971/4a450b6596d7bc15bd01780eb9a607ac/group", "/etc/group").
			Tmpfs("/var/run/nscd", 8192),
	},
}

// fs methods are not implemented using a real FS
// to help better understand filesystem access behaviour
type stubNixOS struct {
	lookPathErr map[string]error
	usernameErr map[string]error
}

func (s *stubNixOS) Geteuid() int {
	return 1971
}

func (s *stubNixOS) LookupEnv(key string) (string, bool) {
	switch key {
	case "SHELL":
		return "/run/current-system/sw/bin/zsh", true
	case "TERM":
		return "xterm-256color", true
	default:
		panic(fmt.Sprintf("attempted to access unexpected environment variable %q", key))
	}
}

func (s *stubNixOS) TempDir() string {
	return "/tmp"
}

func (s *stubNixOS) LookPath(file string) (string, error) {
	if s.lookPathErr != nil {
		if err, ok := s.lookPathErr[file]; ok {
			return "", err
		}
	}

	switch file {
	case "sudo":
		return "/run/wrappers/bin/sudo", nil
	default:
		panic(fmt.Sprintf("attempted to look up unexpected executable %q", file))
	}
}

func (s *stubNixOS) Executable() (string, error) {
	return "/home/ophestra/.nix-profile/bin/fortify", nil
}

func (s *stubNixOS) Lookup(username string) (*user.User, error) {
	if s.usernameErr != nil {
		if err, ok := s.usernameErr[username]; ok {
			return nil, err
		}
	}

	switch username {
	case "chronos":
		return &user.User{
			Uid:      "150",
			Gid:      "101",
			Username: "chronos",
			HomeDir:  "/home/chronos",
		}, nil
	default:
		return nil, user.UnknownUserError(username)
	}
}

func (s *stubNixOS) ReadDir(name string) ([]fs.DirEntry, error) {
	switch name {
	case "/":
		return stubDirEntries("bin", "boot", "dev", "etc", "home", "lib",
			"lib64", "nix", "proc", "root", "run", "srv", "sys", "tmp", "usr", "var")
	case "/run":
		return stubDirEntries("agetty.reload", "binfmt", "booted-system",
			"credentials", "cryptsetup", "current-system", "dbus", "host", "keys",
			"libvirt", "libvirtd.pid", "lock", "log", "lvm", "mount", "NetworkManager",
			"nginx", "nixos", "nscd", "opengl-driver", "pppd", "resolvconf", "sddm",
			"store", "syncoid", "system", "systemd", "tmpfiles.d", "udev", "udisks2",
			"user", "utmp", "virtlogd.pid", "wrappers", "zed.pid", "zed.state")
	case "/etc":
		return stubDirEntries("alsa", "bashrc", "binfmt.d", "dbus-1", "default",
			"ethertypes", "fonts", "fstab", "fuse.conf", "group", "host.conf", "hostid",
			"hostname", "hostname.CHECKSUM", "hosts", "inputrc", "ipsec.d", "issue", "kbd",
			"libblockdev", "locale.conf", "localtime", "login.defs", "lsb-release", "lvm",
			"machine-id", "man_db.conf", "modprobe.d", "modules-load.d", "mtab", "nanorc",
			"netgroup", "NetworkManager", "nix", "nixos", "NIXOS", "nscd.conf", "nsswitch.conf",
			"opensnitchd", "os-release", "pam", "pam.d", "passwd", "pipewire", "pki", "polkit-1",
			"profile", "protocols", "qemu", "resolv.conf", "resolvconf.conf", "rpc", "samba",
			"sddm.conf", "secureboot", "services", "set-environment", "shadow", "shells", "ssh",
			"ssl", "static", "subgid", "subuid", "sudoers", "sysctl.d", "systemd", "terminfo",
			"tmpfiles.d", "udev", "udisks2", "UPower", "vconsole.conf", "X11", "zfs", "zinputrc",
			"zoneinfo", "zprofile", "zshenv", "zshrc")
	default:
		panic(fmt.Sprintf("attempted to read unexpected directory %q", name))
	}
}

func (s *stubNixOS) Stat(name string) (fs.FileInfo, error) {
	switch name {
	case "/var/run/nscd":
		return nil, nil
	default:
		panic(fmt.Sprintf("attempted to stat unexpected path %q", name))
	}
}

func (s *stubNixOS) Open(name string) (fs.File, error) {
	switch name {
	default:
		panic(fmt.Sprintf("attempted to open unexpected file %q", name))
	}
}

func (s *stubNixOS) Exit(code int) {
	panic("called exit on stub with code " + strconv.Itoa(code))
}

func (s *stubNixOS) Paths() internal.Paths {
	return internal.Paths{
		SharePath:   "/tmp/fortify.1971",
		RuntimePath: "/run/user/1971",
		RunDirPath:  "/run/user/1971/fortify",
	}
}
