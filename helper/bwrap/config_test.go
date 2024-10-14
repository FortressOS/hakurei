package bwrap

import (
	"slices"
	"testing"
)

func TestConfig_Args(t *testing.T) {
	testCases := []struct {
		name string
		conf *Config
		want []string
	}{
		{
			name: "xdg-dbus-proxy constraint sample",
			conf: (&Config{
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
				"--unshare-all",
				"--unshare-user",
				"--disable-userns",
				"--assert-userns-disabled",
				"--clearenv",
				"--die-with-parent",
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.conf.Args(); !slices.Equal(got, tc.want) {
				t.Errorf("Args() = %#v, want %#v", got, tc.want)
			}
		})
	}
}
