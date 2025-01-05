package bwrap_test

import (
	"slices"
	"testing"

	"git.gensokyo.uk/security/fortify/helper/bwrap"
)

func TestConfig_Args(t *testing.T) {
	testCases := []struct {
		name string
		conf *bwrap.Config
		want []string
	}{
		{
			name: "overlayfs",
			conf: (new(bwrap.Config)).
				Overlay("/etc", "/etc").
				Join("/.fortify/bin", "/bin", "/usr/bin", "/usr/local/bin").
				Persist("/nix", "/data/data/org.chromium.Chromium/overlay/rwsrc", "/data/data/org.chromium.Chromium/workdir", "/data/app/org.chromium.Chromium/nix"),
			want: []string{
				"--unshare-all", "--unshare-user",
				"--disable-userns", "--assert-userns-disabled",
				// Overlay("/etc", "/etc")
				"--overlay-src", "/etc", "--tmp-overlay", "/etc",
				// Join("/.fortify/bin", "/bin", "/usr/bin", "/usr/local/bin")
				"--overlay-src", "/bin", "--overlay-src", "/usr/bin",
				"--overlay-src", "/usr/local/bin", "--ro-overlay", "/.fortify/bin",
				// Persist("/nix", "/data/data/org.chromium.Chromium/overlay/rwsrc", "/data/data/org.chromium.Chromium/workdir", "/data/app/org.chromium.Chromium/nix")
				"--overlay-src", "/data/app/org.chromium.Chromium/nix",
				"--overlay", "/data/data/org.chromium.Chromium/overlay/rwsrc", "/data/data/org.chromium.Chromium/workdir", "/nix",
			},
		},
		{
			name: "xdg-dbus-proxy constraint sample",
			conf: (&bwrap.Config{
				Unshare:       nil,
				UserNS:        false,
				Clearenv:      true,
				DieWithParent: true,
			}).
				Symlink("usr/bin", "/bin").
				Symlink("var/home", "/home").
				Symlink("usr/lib", "/lib").
				Symlink("usr/lib64", "/lib64").
				Symlink("run/media", "/media").
				Symlink("var/mnt", "/mnt").
				Symlink("var/opt", "/opt").
				Symlink("sysroot/ostree", "/ostree").
				Symlink("var/roothome", "/root").
				Symlink("usr/sbin", "/sbin").
				Symlink("var/srv", "/srv").
				Bind("/run", "/run", false, true).
				Bind("/tmp", "/tmp", false, true).
				Bind("/var", "/var", false, true).
				Bind("/run/user/1971/.dbus-proxy/", "/run/user/1971/.dbus-proxy/", false, true).
				Bind("/boot", "/boot").
				Bind("/dev", "/dev").
				Bind("/proc", "/proc").
				Bind("/sys", "/sys").
				Bind("/sysroot", "/sysroot").
				Bind("/usr", "/usr").
				Bind("/etc", "/etc"),
			want: []string{
				"--unshare-all", "--unshare-user",
				"--disable-userns", "--assert-userns-disabled",
				"--clearenv", "--die-with-parent",
				"--symlink", "usr/bin", "/bin",
				"--symlink", "var/home", "/home",
				"--symlink", "usr/lib", "/lib",
				"--symlink", "usr/lib64", "/lib64",
				"--symlink", "run/media", "/media",
				"--symlink", "var/mnt", "/mnt",
				"--symlink", "var/opt", "/opt",
				"--symlink", "sysroot/ostree", "/ostree",
				"--symlink", "var/roothome", "/root",
				"--symlink", "usr/sbin", "/sbin",
				"--symlink", "var/srv", "/srv",
				"--bind", "/run", "/run",
				"--bind", "/tmp", "/tmp",
				"--bind", "/var", "/var",
				"--bind", "/run/user/1971/.dbus-proxy/", "/run/user/1971/.dbus-proxy/",
				"--ro-bind", "/boot", "/boot",
				"--ro-bind", "/dev", "/dev",
				"--ro-bind", "/proc", "/proc",
				"--ro-bind", "/sys", "/sys",
				"--ro-bind", "/sysroot", "/sysroot",
				"--ro-bind", "/usr", "/usr",
				"--ro-bind", "/etc", "/etc",
			},
		},
		{
			name: "fortify permissive default nixos",
			conf: (&bwrap.Config{
				Unshare:  nil,
				Net:      true,
				UserNS:   true,
				Clearenv: true,
				SetEnv: map[string]string{
					"HOME":              "/home/chronos",
					"TERM":              "xterm-256color",
					"FORTIFY_INIT":      "3",
					"XDG_RUNTIME_DIR":   "/run/user/150",
					"XDG_SESSION_CLASS": "user",
					"XDG_SESSION_TYPE":  "tty",
					"SHELL":             "/run/current-system/sw/bin/zsh",
					"USER":              "chronos",
				},
				DieWithParent: true,
				AsInit:        true,
			}).SetUID(65534).SetGID(65534).
				Procfs("/proc").DevTmpfs("/dev").Mqueue("/dev/mqueue").
				Bind("/bin", "/bin", false, true).
				Bind("/boot", "/boot", false, true).
				Bind("/etc", "/etc", false, true).
				Bind("/home", "/home", false, true).
				Bind("/lib", "/lib", false, true).
				Bind("/lib64", "/lib64", false, true).
				Bind("/nix", "/nix", false, true).
				Bind("/root", "/root", false, true).
				Bind("/srv", "/srv", false, true).
				Bind("/sys", "/sys", false, true).
				Bind("/usr", "/usr", false, true).
				Bind("/var", "/var", false, true).
				Bind("/run/NetworkManager", "/run/NetworkManager", false, true).
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
				Bind("/run/nginx", "/run/nginx", false, true).
				Bind("/run/nscd", "/run/nscd", false, true).
				Bind("/run/opengl-driver", "/run/opengl-driver", false, true).
				Bind("/run/pppd", "/run/pppd", false, true).
				Bind("/run/resolvconf", "/run/resolvconf", false, true).
				Bind("/run/sddm", "/run/sddm", false, true).
				Bind("/run/syncoid", "/run/syncoid", false, true).
				Bind("/run/systemd", "/run/systemd", false, true).
				Bind("/run/tmpfiles.d", "/run/tmpfiles.d", false, true).
				Bind("/run/udev", "/run/udev", false, true).
				Bind("/run/udisks2", "/run/udisks2", false, true).
				Bind("/run/utmp", "/run/utmp", false, true).
				Bind("/run/virtlogd.pid", "/run/virtlogd.pid", false, true).
				Bind("/run/wrappers", "/run/wrappers", false, true).
				Bind("/run/zed.pid", "/run/zed.pid", false, true).
				Bind("/run/zed.state", "/run/zed.state", false, true).
				Bind("/tmp/fortify.1971/tmpdir/150", "/tmp", false, true).
				Tmpfs("/tmp/fortify.1971", 1048576).
				Tmpfs("/run/user", 1048576).
				Tmpfs("/run/user/150", 8388608).
				Bind("/tmp/fortify.1971/67a97cc824a64ef789f16b20ca6ce311/passwd", "/tmp/fortify.1971/67a97cc824a64ef789f16b20ca6ce311/passwd").
				Bind("/tmp/fortify.1971/67a97cc824a64ef789f16b20ca6ce311/group", "/tmp/fortify.1971/67a97cc824a64ef789f16b20ca6ce311/group").
				Bind("/tmp/fortify.1971/67a97cc824a64ef789f16b20ca6ce311/passwd", "/etc/passwd").
				Bind("/tmp/fortify.1971/67a97cc824a64ef789f16b20ca6ce311/group", "/etc/group").
				Tmpfs("/var/run/nscd", 8192),
			want: []string{
				"--unshare-all", "--unshare-user", "--share-net",
				"--clearenv", "--die-with-parent", "--as-pid-1",
				"--uid", "65534",
				"--gid", "65534",
				"--setenv", "FORTIFY_INIT", "3",
				"--setenv", "HOME", "/home/chronos",
				"--setenv", "SHELL", "/run/current-system/sw/bin/zsh",
				"--setenv", "TERM", "xterm-256color",
				"--setenv", "USER", "chronos",
				"--setenv", "XDG_RUNTIME_DIR", "/run/user/150",
				"--setenv", "XDG_SESSION_CLASS", "user",
				"--setenv", "XDG_SESSION_TYPE", "tty",
				"--proc", "/proc", "--dev", "/dev",
				"--mqueue", "/dev/mqueue",
				"--bind", "/bin", "/bin",
				"--bind", "/boot", "/boot",
				"--bind", "/etc", "/etc",
				"--bind", "/home", "/home",
				"--bind", "/lib", "/lib",
				"--bind", "/lib64", "/lib64",
				"--bind", "/nix", "/nix",
				"--bind", "/root", "/root",
				"--bind", "/srv", "/srv",
				"--bind", "/sys", "/sys",
				"--bind", "/usr", "/usr",
				"--bind", "/var", "/var",
				"--bind", "/run/NetworkManager", "/run/NetworkManager",
				"--bind", "/run/agetty.reload", "/run/agetty.reload",
				"--bind", "/run/binfmt", "/run/binfmt",
				"--bind", "/run/booted-system", "/run/booted-system",
				"--bind", "/run/credentials", "/run/credentials",
				"--bind", "/run/cryptsetup", "/run/cryptsetup",
				"--bind", "/run/current-system", "/run/current-system",
				"--bind", "/run/host", "/run/host",
				"--bind", "/run/keys", "/run/keys",
				"--bind", "/run/libvirt", "/run/libvirt",
				"--bind", "/run/libvirtd.pid", "/run/libvirtd.pid",
				"--bind", "/run/lock", "/run/lock",
				"--bind", "/run/log", "/run/log",
				"--bind", "/run/lvm", "/run/lvm",
				"--bind", "/run/mount", "/run/mount",
				"--bind", "/run/nginx", "/run/nginx",
				"--bind", "/run/nscd", "/run/nscd",
				"--bind", "/run/opengl-driver", "/run/opengl-driver",
				"--bind", "/run/pppd", "/run/pppd",
				"--bind", "/run/resolvconf", "/run/resolvconf",
				"--bind", "/run/sddm", "/run/sddm",
				"--bind", "/run/syncoid", "/run/syncoid",
				"--bind", "/run/systemd", "/run/systemd",
				"--bind", "/run/tmpfiles.d", "/run/tmpfiles.d",
				"--bind", "/run/udev", "/run/udev",
				"--bind", "/run/udisks2", "/run/udisks2",
				"--bind", "/run/utmp", "/run/utmp",
				"--bind", "/run/virtlogd.pid", "/run/virtlogd.pid",
				"--bind", "/run/wrappers", "/run/wrappers",
				"--bind", "/run/zed.pid", "/run/zed.pid",
				"--bind", "/run/zed.state", "/run/zed.state",
				"--bind", "/tmp/fortify.1971/tmpdir/150", "/tmp",
				"--size", "1048576", "--tmpfs", "/tmp/fortify.1971",
				"--size", "1048576", "--tmpfs", "/run/user",
				"--size", "8388608", "--tmpfs", "/run/user/150",
				"--ro-bind", "/tmp/fortify.1971/67a97cc824a64ef789f16b20ca6ce311/passwd", "/tmp/fortify.1971/67a97cc824a64ef789f16b20ca6ce311/passwd",
				"--ro-bind", "/tmp/fortify.1971/67a97cc824a64ef789f16b20ca6ce311/group", "/tmp/fortify.1971/67a97cc824a64ef789f16b20ca6ce311/group",
				"--ro-bind", "/tmp/fortify.1971/67a97cc824a64ef789f16b20ca6ce311/passwd", "/etc/passwd",
				"--ro-bind", "/tmp/fortify.1971/67a97cc824a64ef789f16b20ca6ce311/group", "/etc/group",
				"--size", "8192", "--tmpfs", "/var/run/nscd",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.conf.Args(); !slices.Equal(got, tc.want) {
				t.Errorf("Args() = %#v, want %#v", got, tc.want)
			}
		})
	}
}
