package app

import (
	"fmt"
	"io/fs"
	"log"
	"os/exec"
	"os/user"
)

type stubNixOS struct {
	lookPathErr map[string]error
	usernameErr map[string]error
}

func (k *stubNixOS) new(func(k syscallDispatcher)) { panic("not implemented") }

func (k *stubNixOS) getuid() int { return 1971 }
func (k *stubNixOS) getgid() int { return 100 }

func (k *stubNixOS) lookupEnv(key string) (string, bool) {
	switch key {
	case "SHELL":
		return "/run/current-system/sw/bin/zsh", true
	case "TERM":
		return "xterm-256color", true
	case "WAYLAND_DISPLAY":
		return "wayland-0", true
	case "PULSE_COOKIE":
		return "", false
	case "HOME":
		return "/home/ophestra", true
	case "XDG_RUNTIME_DIR":
		return "/run/user/1971", true
	case "XDG_CONFIG_HOME":
		return "/home/ophestra/xdg/config", true
	default:
		panic(fmt.Sprintf("attempted to access unexpected environment variable %q", key))
	}
}

func (k *stubNixOS) stat(name string) (fs.FileInfo, error) {
	switch name {
	case "/var/run/nscd":
		return nil, nil
	case "/run/user/1971/pulse":
		return nil, nil
	case "/run/user/1971/pulse/native":
		return stubFileInfoMode(0666), nil
	case "/home/ophestra/.pulse-cookie":
		return stubFileInfoIsDir(true), nil
	case "/home/ophestra/xdg/config/pulse/cookie":
		return stubFileInfoIsDir(false), nil
	default:
		panic(fmt.Sprintf("attempted to stat unexpected path %q", name))
	}
}

func (k *stubNixOS) readdir(name string) ([]fs.DirEntry, error) {
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

func (k *stubNixOS) tempdir() string { return "/tmp/" }

func (k *stubNixOS) evalSymlinks(path string) (string, error) {
	switch path {
	case "/run/user/1971":
		return "/run/user/1971", nil
	case "/tmp/hakurei.0":
		return "/tmp/hakurei.0", nil
	case "/run/dbus":
		return "/run/dbus", nil
	case "/dev/kvm":
		return "/dev/kvm", nil
	case "/etc/":
		return "/etc/", nil
	case "/bin":
		return "/bin", nil
	case "/boot":
		return "/boot", nil
	case "/home":
		return "/home", nil
	case "/lib":
		return "/lib", nil
	case "/lib64":
		return "/lib64", nil
	case "/nix":
		return "/nix", nil
	case "/root":
		return "/root", nil
	case "/run":
		return "/run", nil
	case "/srv":
		return "/srv", nil
	case "/sys":
		return "/sys", nil
	case "/usr":
		return "/usr", nil
	case "/var":
		return "/var", nil
	case "/dev/dri":
		return "/dev/dri", nil
	case "/usr/bin/":
		return "/usr/bin/", nil
	case "/nix/store":
		return "/nix/store", nil
	case "/run/current-system":
		return "/nix/store/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-nixos-system-satori-25.05.99999999.aaaaaaa", nil
	case "/sys/block":
		return "/sys/block", nil
	case "/sys/bus":
		return "/sys/bus", nil
	case "/sys/class":
		return "/sys/class", nil
	case "/sys/dev":
		return "/sys/dev", nil
	case "/sys/devices":
		return "/sys/devices", nil
	case "/run/opengl-driver":
		return "/nix/store/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-graphics-drivers", nil
	case "/var/lib/persist/module/hakurei/0/1":
		return "/var/lib/persist/module/hakurei/0/1", nil
	default:
		panic(fmt.Sprintf("attempted to evaluate unexpected path %q", path))
	}
}

func (k *stubNixOS) lookPath(file string) (string, error) {
	if k.lookPathErr != nil {
		if err, ok := k.lookPathErr[file]; ok {
			return "", err
		}
	}

	switch file {
	case "zsh":
		return "/run/current-system/sw/bin/zsh", nil
	default:
		panic(fmt.Sprintf("attempted to look up unexpected executable %q", file))
	}
}

func (k *stubNixOS) lookupGroupId(name string) (string, error) {
	switch name {
	case "video":
		return "26", nil
	default:
		return "", user.UnknownGroupError(name)
	}
}

func (k *stubNixOS) cmdOutput(cmd *exec.Cmd) ([]byte, error) {
	switch cmd.Path {
	case "/proc/nonexistent/hsu":
		return []byte{'0'}, nil
	default:
		panic(fmt.Sprintf("unexpected cmd %#v", cmd))
	}
}

func (k *stubNixOS) overflowUid() int { return 65534 }
func (k *stubNixOS) overflowGid() int { return 65534 }

func (k *stubNixOS) mustHsuPath() string { return "/proc/nonexistent/hsu" }

func (k *stubNixOS) fatalf(format string, v ...any) { panic(fmt.Sprintf(format, v...)) }

func (k *stubNixOS) isVerbose() bool                  { return true }
func (k *stubNixOS) verbose(v ...any)                 { log.Print(v...) }
func (k *stubNixOS) verbosef(format string, v ...any) { log.Printf(format, v...) }
