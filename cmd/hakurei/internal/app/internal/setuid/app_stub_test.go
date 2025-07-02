package setuid_test

import (
	"fmt"
	"io/fs"
	"log"
	"os/user"
	"strconv"

	"hakurei.app/hst"
)

// fs methods are not implemented using a real FS
// to help better understand filesystem access behaviour
type stubNixOS struct {
	lookPathErr map[string]error
	usernameErr map[string]error
}

func (s *stubNixOS) Getuid() int                              { return 1971 }
func (s *stubNixOS) Getgid() int                              { return 100 }
func (s *stubNixOS) TempDir() string                          { return "/tmp" }
func (s *stubNixOS) MustExecutable() string                   { return "/run/wrappers/bin/hakurei" }
func (s *stubNixOS) Exit(code int)                            { panic("called exit on stub with code " + strconv.Itoa(code)) }
func (s *stubNixOS) EvalSymlinks(path string) (string, error) { return path, nil }
func (s *stubNixOS) Uid(aid int) (int, error)                 { return 1000000 + 0*10000 + aid, nil }

func (s *stubNixOS) Println(v ...any)               { log.Println(v...) }
func (s *stubNixOS) Printf(format string, v ...any) { log.Printf(format, v...) }

func (s *stubNixOS) LookupEnv(key string) (string, bool) {
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
	case "XDG_CONFIG_HOME":
		return "/home/ophestra/xdg/config", true
	default:
		panic(fmt.Sprintf("attempted to access unexpected environment variable %q", key))
	}
}

func (s *stubNixOS) LookPath(file string) (string, error) {
	if s.lookPathErr != nil {
		if err, ok := s.lookPathErr[file]; ok {
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

func (s *stubNixOS) LookupGroup(name string) (*user.Group, error) {
	switch name {
	case "video":
		return &user.Group{Gid: "26", Name: "video"}, nil
	default:
		return nil, user.UnknownGroupError(name)
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

func (s *stubNixOS) Open(name string) (fs.File, error) {
	switch name {
	default:
		panic(fmt.Sprintf("attempted to open unexpected file %q", name))
	}
}

func (s *stubNixOS) Paths() hst.Paths {
	return hst.Paths{
		SharePath:   "/tmp/hakurei.1971",
		RuntimePath: "/run/user/1971",
		RunDirPath:  "/run/user/1971/hakurei",
	}
}
