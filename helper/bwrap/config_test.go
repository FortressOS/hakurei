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
			conf: &Config{
				Unshare:  nil,
				UserNS:   false,
				Clearenv: true,
				Symlink: []PermConfig[[2]string]{
					{Path: [2]string{"usr/bin", "/bin"}},
					{Path: [2]string{"var/home", "/home"}},
					{Path: [2]string{"usr/lib", "/lib"}},
					{Path: [2]string{"usr/lib64", "/lib64"}},
					{Path: [2]string{"run/media", "/media"}},
					{Path: [2]string{"var/mnt", "/mnt"}},
					{Path: [2]string{"var/opt", "/opt"}},
					{Path: [2]string{"sysroot/ostree", "/ostree"}},
					{Path: [2]string{"var/roothome", "/root"}},
					{Path: [2]string{"usr/sbin", "/sbin"}},
					{Path: [2]string{"var/srv", "/srv"}},
				},
				Bind: [][2]string{
					{"/run", "/run"},
					{"/tmp", "/tmp"},
					{"/var", "/var"},
					{"/run/user/1971/.dbus-proxy/", "/run/user/1971/.dbus-proxy/"},
				},
				ROBind: [][2]string{
					{"/boot", "/boot"},
					{"/dev", "/dev"},
					{"/proc", "/proc"},
					{"/sys", "/sys"},
					{"/sysroot", "/sysroot"},
					{"/usr", "/usr"},
					{"/etc", "/etc"},
				},
				DieWithParent: true,
			},
			want: []string{
				"--unshare-all",
				"--disable-userns",
				"--assert-userns-disabled",
				"--clearenv",
				"--die-with-parent",
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
				"--symlink", "var/srv", "/srv"},
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
