package container_test

import (
	"bytes"
	"context"
	"encoding/gob"
	"log"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"hakurei.app/container"
	"hakurei.app/container/seccomp"
	"hakurei.app/container/vfs"
	"hakurei.app/hst"
	"hakurei.app/internal"
	"hakurei.app/internal/hlog"
	"hakurei.app/ldd"
)

const (
	ignore  = "\x00"
	ignoreV = -1
)

func TestMain(m *testing.M) {
	container.TryArgv0(hlog.Output{}, hlog.Prepare, internal.InstallOutput)
	os.Exit(m.Run())
}

func TestContainer(t *testing.T) {
	{
		oldVerbose := hlog.Load()
		oldOutput := container.GetOutput()
		internal.InstallOutput(true)
		t.Cleanup(func() { hlog.Store(oldVerbose) })
		t.Cleanup(func() { container.SetOutput(oldOutput) })
	}

	testCases := []struct {
		name    string
		filter  bool
		session bool
		net     bool
		ops     *container.Ops
		mnt     []*vfs.MountInfoEntry
		host    string
		rules   []seccomp.NativeRule
		flags   seccomp.ExportFlag
		presets seccomp.FilterPreset
	}{
		{"minimal", true, false, false,
			new(container.Ops), nil, "test-minimal",
			nil, 0, seccomp.PresetStrict},
		{"allow", true, true, true,
			new(container.Ops), nil, "test-minimal",
			nil, 0, seccomp.PresetExt | seccomp.PresetDenyDevel},
		{"no filter", false, true, true,
			new(container.Ops), nil, "test-no-filter",
			nil, 0, seccomp.PresetExt},
		{"custom rules", true, true, true,
			new(container.Ops), nil, "test-no-filter",
			[]seccomp.NativeRule{
				{seccomp.ScmpSyscall(syscall.SYS_SETUID), seccomp.ScmpErrno(syscall.EPERM), nil},
			}, 0, seccomp.PresetExt},
		{"tmpfs", true, false, false,
			new(container.Ops).
				Tmpfs(hst.Tmp, 0, 0755),
			[]*vfs.MountInfoEntry{
				e("/", hst.Tmp, "rw,nosuid,nodev,relatime", "tmpfs", "tmpfs", ignore),
			}, "test-tmpfs",
			nil, 0, seccomp.PresetStrict},
		{"dev", true, true /* go test output is not a tty */, false,
			new(container.Ops).
				Dev("/dev").
				Mqueue("/dev/mqueue"),
			[]*vfs.MountInfoEntry{
				e("/", "/dev", "rw,nosuid,nodev,relatime", "tmpfs", "devtmpfs", ignore),
				e("/null", "/dev/null", "rw,nosuid", "devtmpfs", "devtmpfs", ignore),
				e("/zero", "/dev/zero", "rw,nosuid", "devtmpfs", "devtmpfs", ignore),
				e("/full", "/dev/full", "rw,nosuid", "devtmpfs", "devtmpfs", ignore),
				e("/random", "/dev/random", "rw,nosuid", "devtmpfs", "devtmpfs", ignore),
				e("/urandom", "/dev/urandom", "rw,nosuid", "devtmpfs", "devtmpfs", ignore),
				e("/tty", "/dev/tty", "rw,nosuid", "devtmpfs", "devtmpfs", ignore),
				e("/", "/dev/pts", "rw,nosuid,noexec,relatime", "devpts", "devpts", "rw,mode=620,ptmxmode=666"),
				e("/", "/dev/mqueue", "rw,nosuid,nodev,noexec,relatime", "mqueue", "mqueue", "rw"),
			}, "",
			nil, 0, seccomp.PresetStrict},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()

			c := container.New(ctx, "/usr/bin/sandbox.test", "-test.v",
				"-test.run=TestHelperCheckContainer", "--", "check", tc.host)
			c.Uid = 1000
			c.Gid = 100
			c.Hostname = tc.host
			c.Stdout, c.Stderr = os.Stdout, os.Stderr
			c.Ops = tc.ops
			c.SeccompRules = tc.rules
			c.SeccompFlags = tc.flags | seccomp.AllowMultiarch
			c.SeccompPresets = tc.presets
			c.SeccompDisable = !tc.filter
			c.RetainSession = tc.session
			c.HostNet = tc.net
			if c.Args[5] == "" {
				if name, err := os.Hostname(); err != nil {
					t.Fatalf("cannot get hostname: %v", err)
				} else {
					c.Args[5] = name
				}
			}

			c.
				Tmpfs("/tmp", 0, 0755).
				Bind(os.Args[0], os.Args[0], 0).
				Mkdir("/usr/bin", 0755).
				Link(os.Args[0], "/usr/bin/sandbox.test").
				Place("/etc/hostname", []byte(c.Args[5]))
			// in case test has cgo enabled
			var libPaths []string
			if entries, err := ldd.Exec(ctx, os.Args[0]); err != nil {
				log.Fatalf("ldd: %v", err)
			} else {
				libPaths = ldd.Path(entries)
			}
			for _, name := range libPaths {
				c.Bind(name, name, 0)
			}
			// needs /proc to check mountinfo
			c.Proc("/proc")

			mnt := make([]*vfs.MountInfoEntry, 0, 3+len(libPaths))
			mnt = append(mnt, e("/sysroot", "/", "rw,nosuid,nodev,relatime", "tmpfs", "rootfs", ignore))
			mnt = append(mnt, tc.mnt...)
			mnt = append(mnt,
				e("/", "/tmp", "rw,nosuid,nodev,relatime", "tmpfs", "tmpfs", ignore),
				e(ignore, os.Args[0], "ro,nosuid,nodev,relatime", ignore, ignore, ignore),
				e(ignore, "/etc/hostname", "ro,nosuid,nodev,relatime", "tmpfs", "rootfs", ignore),
			)
			for _, name := range libPaths {
				mnt = append(mnt, e(ignore, name, "ro,nosuid,nodev,relatime", ignore, ignore, ignore))
			}
			mnt = append(mnt, e("/", "/proc", "rw,nosuid,nodev,noexec,relatime", "proc", "proc", "rw"))
			want := new(bytes.Buffer)
			if err := gob.NewEncoder(want).Encode(mnt); err != nil {
				t.Fatalf("cannot serialise expected mount points: %v", err)
			}
			c.Stdin = want

			if err := c.Start(); err != nil {
				hlog.PrintBaseError(err, "start:")
				t.Fatalf("cannot start container: %v", err)
			} else if err = c.Serve(); err != nil {
				hlog.PrintBaseError(err, "serve:")
				t.Errorf("cannot serve setup params: %v", err)
			}
			if err := c.Wait(); err != nil {
				hlog.PrintBaseError(err, "wait:")
				t.Fatalf("wait: %v", err)
			}
		})
	}
}

func e(root, target, vfsOptstr, fsType, source, fsOptstr string) *vfs.MountInfoEntry {
	return &vfs.MountInfoEntry{
		ID:        ignoreV,
		Parent:    ignoreV,
		Devno:     vfs.DevT{ignoreV, ignoreV},
		Root:      root,
		Target:    target,
		VfsOptstr: vfsOptstr,
		OptFields: []string{ignore},
		FsType:    fsType,
		Source:    source,
		FsOptstr:  fsOptstr,
	}
}

func TestContainerString(t *testing.T) {
	c := container.New(t.Context(), "ldd", "/usr/bin/env")
	c.SeccompFlags |= seccomp.AllowMultiarch
	c.SeccompRules = seccomp.Preset(
		seccomp.PresetExt|seccomp.PresetDenyNS|seccomp.PresetDenyTTY,
		c.SeccompFlags)
	c.SeccompPresets = seccomp.PresetStrict
	want := `argv: ["ldd" "/usr/bin/env"], filter: true, rules: 65, flags: 0x1, presets: 0xf`
	if got := c.String(); got != want {
		t.Errorf("String: %s, want %s", got, want)
	}
}

func TestHelperCheckContainer(t *testing.T) {
	if len(os.Args) != 6 || os.Args[4] != "check" {
		return
	}

	t.Run("user", func(t *testing.T) {
		if uid := syscall.Getuid(); uid != 1000 {
			t.Errorf("Getuid: %d, want 1000", uid)
		}
		if gid := syscall.Getgid(); gid != 100 {
			t.Errorf("Getgid: %d, want 100", gid)
		}
	})
	t.Run("hostname", func(t *testing.T) {
		if name, err := os.Hostname(); err != nil {
			t.Fatalf("cannot get hostname: %v", err)
		} else if name != os.Args[5] {
			t.Errorf("Hostname: %q, want %q", name, os.Args[5])
		}

		if p, err := os.ReadFile("/etc/hostname"); err != nil {
			t.Fatalf("%v", err)
		} else if string(p) != os.Args[5] {
			t.Errorf("/etc/hostname: %q, want %q", string(p), os.Args[5])
		}
	})
	t.Run("mount", func(t *testing.T) {
		var mnt []*vfs.MountInfoEntry
		if err := gob.NewDecoder(os.Stdin).Decode(&mnt); err != nil {
			t.Fatalf("cannot receive expected mount points: %v", err)
		}

		var d *vfs.MountInfoDecoder
		if f, err := os.Open("/proc/self/mountinfo"); err != nil {
			t.Fatalf("cannot open mountinfo: %v", err)
		} else {
			d = vfs.NewMountInfoDecoder(f)
		}

		i := 0
		for cur := range d.Entries() {
			if i == len(mnt) {
				t.Errorf("got more than %d entries", len(mnt))
				break
			}

			// ugly hack but should be reliable and is less likely to false negative than comparing by parsed flags
			cur.VfsOptstr = strings.TrimSuffix(cur.VfsOptstr, ",relatime")
			cur.VfsOptstr = strings.TrimSuffix(cur.VfsOptstr, ",noatime")
			mnt[i].VfsOptstr = strings.TrimSuffix(mnt[i].VfsOptstr, ",relatime")
			mnt[i].VfsOptstr = strings.TrimSuffix(mnt[i].VfsOptstr, ",noatime")

			if !cur.EqualWithIgnore(mnt[i], "\x00") {
				t.Errorf("[FAIL] %s", cur)
			} else {
				t.Logf("[ OK ] %s", cur)
			}

			i++
		}
		if err := d.Err(); err != nil {
			t.Errorf("cannot parse mountinfo: %v", err)
		}

		if i != len(mnt) {
			t.Errorf("got %d entries, want %d", i, len(mnt))
		}
	})
}
