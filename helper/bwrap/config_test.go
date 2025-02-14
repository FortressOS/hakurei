package bwrap_test

import (
	"os"
	"slices"
	"testing"

	"git.gensokyo.uk/security/fortify/helper/bwrap"
	"git.gensokyo.uk/security/fortify/helper/proc"
	"git.gensokyo.uk/security/fortify/helper/seccomp"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
)

func TestConfig_Args(t *testing.T) {
	seccomp.CPrintln = fmsg.Println
	t.Cleanup(func() { seccomp.CPrintln = nil })

	testCases := []struct {
		name string
		conf *bwrap.Config
		want []string
	}{
		{
			"bind", (new(bwrap.Config)).
				Bind("/etc", "/.fortify/etc").
				Bind("/etc", "/.fortify/etc", true).
				Bind("/run", "/.fortify/run", false, true).
				Bind("/sys/devices", "/.fortify/sys/devices", true, true).
				Bind("/dev/dri", "/.fortify/dev/dri", false, true, true).
				Bind("/dev/dri", "/.fortify/dev/dri", true, true, true),
			[]string{
				"--unshare-all", "--unshare-user",
				"--disable-userns", "--assert-userns-disabled",
				// Bind("/etc", "/.fortify/etc")
				"--ro-bind", "/etc", "/.fortify/etc",
				// Bind("/etc", "/.fortify/etc", true)
				"--ro-bind-try", "/etc", "/.fortify/etc",
				// Bind("/run", "/.fortify/run", false, true)
				"--bind", "/run", "/.fortify/run",
				// Bind("/sys/devices", "/.fortify/sys/devices", true, true)
				"--bind-try", "/sys/devices", "/.fortify/sys/devices",
				// Bind("/dev/dri", "/.fortify/dev/dri", false, true, true)
				"--dev-bind", "/dev/dri", "/.fortify/dev/dri",
				// Bind("/dev/dri", "/.fortify/dev/dri", true, true, true)
				"--dev-bind-try", "/dev/dri", "/.fortify/dev/dri",
			},
		},
		{
			"dir remount-ro proc dev mqueue", (new(bwrap.Config)).
				Dir("/.fortify").
				RemountRO("/home").
				Procfs("/proc").
				DevTmpfs("/dev").
				Mqueue("/dev/mqueue"),
			[]string{
				"--unshare-all", "--unshare-user",
				"--disable-userns", "--assert-userns-disabled",
				// Dir("/.fortify")
				"--dir", "/.fortify",
				// RemountRO("/home")
				"--remount-ro", "/home",
				// Procfs("/proc")
				"--proc", "/proc",
				// DevTmpfs("/dev")
				"--dev", "/dev",
				// Mqueue("/dev/mqueue")
				"--mqueue", "/dev/mqueue",
			},
		},
		{
			"tmpfs", (new(bwrap.Config)).
				Tmpfs("/run/user", 8192).
				Tmpfs("/run/dbus", 8192, 0755),
			[]string{
				"--unshare-all", "--unshare-user",
				"--disable-userns", "--assert-userns-disabled",
				// Tmpfs("/run/user", 8192)
				"--size", "8192", "--tmpfs", "/run/user",
				// Tmpfs("/run/dbus", 8192, 0755)
				"--perms", "755", "--size", "8192", "--tmpfs", "/run/dbus",
			},
		},
		{
			"symlink", (new(bwrap.Config)).
				Symlink("/.fortify/sbin/init", "/sbin/init").
				Symlink("/.fortify/sbin/init", "/sbin/init", 0755),
			[]string{
				"--unshare-all", "--unshare-user",
				"--disable-userns", "--assert-userns-disabled",
				// Symlink("/.fortify/sbin/init", "/sbin/init")
				"--symlink", "/.fortify/sbin/init", "/sbin/init",
				// Symlink("/.fortify/sbin/init", "/sbin/init", 0755)
				"--perms", "755", "--symlink", "/.fortify/sbin/init", "/sbin/init",
			},
		},
		{
			"overlayfs", (new(bwrap.Config)).
				Overlay("/etc", "/etc").
				Join("/.fortify/bin", "/bin", "/usr/bin", "/usr/local/bin").
				Persist("/nix", "/data/data/org.chromium.Chromium/overlay/rwsrc", "/data/data/org.chromium.Chromium/workdir", "/data/app/org.chromium.Chromium/nix"),
			[]string{
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
			"copy", (new(bwrap.Config)).
				Write("/.fortify/version", make([]byte, 8)).
				CopyBind("/etc/group", make([]byte, 8)).
				CopyBind("/etc/passwd", make([]byte, 8), true),
			[]string{
				"--unshare-all", "--unshare-user",
				"--disable-userns", "--assert-userns-disabled",
				// Write("/.fortify/version", make([]byte, 8))
				"--file", "3", "/.fortify/version",
				// CopyBind("/etc/group", make([]byte, 8))
				"--ro-bind-data", "4", "/etc/group",
				// CopyBind("/etc/passwd", make([]byte, 8), true)
				"--bind-data", "5", "/etc/passwd",
			},
		},
		{
			"unshare", &bwrap.Config{Unshare: &bwrap.UnshareConfig{
				User:   false,
				IPC:    false,
				PID:    false,
				Net:    false,
				UTS:    false,
				CGroup: false,
			}},
			[]string{"--disable-userns", "--assert-userns-disabled"},
		},
		{
			"uid gid sync", (new(bwrap.Config)).
				SetUID(1971).
				SetGID(100),
			[]string{
				"--unshare-all", "--unshare-user",
				"--disable-userns", "--assert-userns-disabled",
				// SetUID(1971)
				"--uid", "1971",
				// SetGID(100)
				"--gid", "100",
			},
		},
		{
			"hostname chdir setenv unsetenv lockfile chmod syscall", &bwrap.Config{
				Hostname: "fortify",
				Chdir:    "/.fortify",
				SetEnv:   map[string]string{"FORTIFY_INIT": "/.fortify/sbin/init"},
				UnsetEnv: []string{"HOME", "HOST"},
				LockFile: []string{"/.fortify/lock"},
				Syscall:  new(bwrap.SyscallPolicy),
				Chmod:    map[string]os.FileMode{"/.fortify/sbin/init": 0755},
			},
			[]string{
				"--unshare-all", "--unshare-user",
				"--disable-userns", "--assert-userns-disabled",
				// Hostname: "fortify"
				"--hostname", "fortify",
				// Chdir: "/.fortify"
				"--chdir", "/.fortify",
				// UnsetEnv: []string{"HOME", "HOST"}
				"--unsetenv", "HOME",
				"--unsetenv", "HOST",
				// LockFile: []string{"/.fortify/lock"},
				"--lock-file", "/.fortify/lock",
				// SetEnv: map[string]string{"FORTIFY_INIT": "/.fortify/sbin/init"}
				"--setenv", "FORTIFY_INIT", "/.fortify/sbin/init",
				// Syscall: new(bwrap.SyscallPolicy),
				"--seccomp", "3",
				// Chmod: map[string]os.FileMode{"/.fortify/sbin/init": 0755}
				"--chmod", "755", "/.fortify/sbin/init",
			},
		},

		{
			"xdg-dbus-proxy constraint sample", (&bwrap.Config{Clearenv: true, DieWithParent: true}).
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
			[]string{
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.conf.Args(nil, new(proc.ExtraFilesPre), new([]proc.File)); !slices.Equal(got, tc.want) {
				t.Errorf("Args() = %#v, want %#v", got, tc.want)
			}
		})
	}

	// test persist validation
	t.Run("invalid persist", func(t *testing.T) {
		defer func() {
			wantPanic := "persist called without required paths"
			if r := recover(); r != wantPanic {
				t.Errorf("Persist() panic = %v; wantPanic %v", r, wantPanic)
			}
		}()
		(new(bwrap.Config)).Persist("/run", "", "")
	})
}
