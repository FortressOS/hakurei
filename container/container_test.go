package container_test

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"hakurei.app/command"
	"hakurei.app/container"
	"hakurei.app/container/seccomp"
	"hakurei.app/container/vfs"
	"hakurei.app/hst"
	"hakurei.app/internal/hlog"
)

const (
	ignore  = "\x00"
	ignoreV = -1

	pathPrefix   = "/etc/hakurei/"
	pathWantMnt  = pathPrefix + "want-mnt"
	pathReadonly = pathPrefix + "readonly"
)

var containerTestCases = []struct {
	name    string
	filter  bool
	session bool
	net     bool
	ro      bool
	ops     *container.Ops

	mnt []*vfs.MountInfoEntry
	uid int
	gid int

	rules   []seccomp.NativeRule
	flags   seccomp.ExportFlag
	presets seccomp.FilterPreset
}{
	{"minimal", true, false, false, true,
		new(container.Ops), nil,
		1000, 100, nil, 0, seccomp.PresetStrict},
	{"allow", true, true, true, false,
		new(container.Ops), nil,
		1000, 100, nil, 0, seccomp.PresetExt | seccomp.PresetDenyDevel},
	{"no filter", false, true, true, true,
		new(container.Ops), nil,
		1000, 100, nil, 0, seccomp.PresetExt},
	{"custom rules", true, true, true, false,
		new(container.Ops), nil,
		1, 31, []seccomp.NativeRule{{seccomp.ScmpSyscall(syscall.SYS_SETUID), seccomp.ScmpErrno(syscall.EPERM), nil}}, 0, seccomp.PresetExt},

	{"tmpfs", true, false, false, true,
		new(container.Ops).
			Tmpfs(hst.Tmp, 0, 0755),
		[]*vfs.MountInfoEntry{
			ent("/", hst.Tmp, "rw,nosuid,nodev,relatime", "tmpfs", "ephemeral", ignore),
		},
		9, 9, nil, 0, seccomp.PresetStrict},

	{"dev", true, true /* go test output is not a tty */, false, false,
		new(container.Ops).
			Dev("/dev", true),
		[]*vfs.MountInfoEntry{
			ent("/", "/dev", "ro,nosuid,nodev,relatime", "tmpfs", "devtmpfs", ignore),
			ent("/null", "/dev/null", "rw,nosuid", "devtmpfs", "devtmpfs", ignore),
			ent("/zero", "/dev/zero", "rw,nosuid", "devtmpfs", "devtmpfs", ignore),
			ent("/full", "/dev/full", "rw,nosuid", "devtmpfs", "devtmpfs", ignore),
			ent("/random", "/dev/random", "rw,nosuid", "devtmpfs", "devtmpfs", ignore),
			ent("/urandom", "/dev/urandom", "rw,nosuid", "devtmpfs", "devtmpfs", ignore),
			ent("/tty", "/dev/tty", "rw,nosuid", "devtmpfs", "devtmpfs", ignore),
			ent("/", "/dev/pts", "rw,nosuid,noexec,relatime", "devpts", "devpts", "rw,mode=620,ptmxmode=666"),
			ent("/", "/dev/mqueue", "rw,nosuid,nodev,noexec,relatime", "mqueue", "mqueue", "rw"),
		},
		1971, 100, nil, 0, seccomp.PresetStrict},

	{"dev no mqueue", true, true /* go test output is not a tty */, false, false,
		new(container.Ops).
			Dev("/dev", false),
		[]*vfs.MountInfoEntry{
			ent("/", "/dev", "ro,nosuid,nodev,relatime", "tmpfs", "devtmpfs", ignore),
			ent("/null", "/dev/null", "rw,nosuid", "devtmpfs", "devtmpfs", ignore),
			ent("/zero", "/dev/zero", "rw,nosuid", "devtmpfs", "devtmpfs", ignore),
			ent("/full", "/dev/full", "rw,nosuid", "devtmpfs", "devtmpfs", ignore),
			ent("/random", "/dev/random", "rw,nosuid", "devtmpfs", "devtmpfs", ignore),
			ent("/urandom", "/dev/urandom", "rw,nosuid", "devtmpfs", "devtmpfs", ignore),
			ent("/tty", "/dev/tty", "rw,nosuid", "devtmpfs", "devtmpfs", ignore),
			ent("/", "/dev/pts", "rw,nosuid,noexec,relatime", "devpts", "devpts", "rw,mode=620,ptmxmode=666"),
		},
		1971, 100, nil, 0, seccomp.PresetStrict},
}

func TestContainer(t *testing.T) {
	{
		oldVerbose := hlog.Load()
		oldOutput := container.GetOutput()
		hlog.Store(testing.Verbose())
		container.SetOutput(hlog.Output{})
		t.Cleanup(func() { hlog.Store(oldVerbose) })
		t.Cleanup(func() { container.SetOutput(oldOutput) })
	}

	t.Run("cancel", testContainerCancel(nil, func(t *testing.T, c *container.Container) {
		wantErr := context.Canceled
		wantExitCode := 0
		if err := c.Wait(); !errors.Is(err, wantErr) {
			hlog.PrintBaseError(err, "wait:")
			t.Errorf("Wait: error = %v, want %v", err, wantErr)
		}
		if ps := c.ProcessState(); ps == nil {
			t.Errorf("ProcessState unexpectedly returned nil")
		} else if code := ps.ExitCode(); code != wantExitCode {
			t.Errorf("ExitCode: %d, want %d", code, wantExitCode)
		}
	}))

	t.Run("forward", testContainerCancel(func(c *container.Container) {
		c.ForwardCancel = true
	}, func(t *testing.T, c *container.Container) {
		var exitError *exec.ExitError
		if err := c.Wait(); !errors.As(err, &exitError) {
			hlog.PrintBaseError(err, "wait:")
			t.Errorf("Wait: error = %v", err)
		}
		if code := exitError.ExitCode(); code != blockExitCodeInterrupt {
			t.Errorf("ExitCode: %d, want %d", code, blockExitCodeInterrupt)
		}
	}))

	for i, tc := range containerTestCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), helperDefaultTimeout)
			defer cancel()

			var libPaths []string
			c := helperNewContainerLibPaths(ctx, &libPaths, "container", strconv.Itoa(i))
			c.Uid = tc.uid
			c.Gid = tc.gid
			c.Hostname = hostnameFromTestCase(tc.name)
			output := new(bytes.Buffer)
			if !testing.Verbose() {
				c.Stdout, c.Stderr = output, output
			} else {
				c.Stdout, c.Stderr = os.Stdout, os.Stderr
			}
			c.WaitDelay = helperDefaultTimeout
			*c.Ops = append(*c.Ops, *tc.ops...)
			c.SeccompRules = tc.rules
			c.SeccompFlags = tc.flags | seccomp.AllowMultiarch
			c.SeccompPresets = tc.presets
			c.SeccompDisable = !tc.filter
			c.RetainSession = tc.session
			c.HostNet = tc.net

			c.
				Readonly(pathReadonly, 0755).
				Tmpfs("/tmp", 0, 0755).
				Place("/etc/hostname", []byte(c.Hostname))
			// needs /proc to check mountinfo
			c.Proc("/proc")

			// mountinfo cannot be resolved directly by helper due to libPaths nondeterminism
			mnt := make([]*vfs.MountInfoEntry, 0, 3+len(libPaths))
			mnt = append(mnt,
				ent("/sysroot", "/", "rw,nosuid,nodev,relatime", "tmpfs", "rootfs", ignore),
				// Bind(os.Args[0], helperInnerPath, 0)
				ent(ignore, helperInnerPath, "ro,nosuid,nodev,relatime", ignore, ignore, ignore),
			)
			for _, name := range libPaths {
				// Bind(name, name, 0)
				mnt = append(mnt, ent(ignore, name, "ro,nosuid,nodev,relatime", ignore, ignore, ignore))
			}
			mnt = append(mnt, tc.mnt...)
			mnt = append(mnt,
				// Readonly(pathReadonly, 0755)
				ent("/", pathReadonly, "ro,nosuid,nodev", "tmpfs", "readonly", ignore),
				// Tmpfs("/tmp", 0, 0755)
				ent("/", "/tmp", "rw,nosuid,nodev,relatime", "tmpfs", "ephemeral", ignore),
				// Place("/etc/hostname", []byte(hostname))
				ent(ignore, "/etc/hostname", "ro,nosuid,nodev,relatime", "tmpfs", "rootfs", ignore),
				// Proc("/proc")
				ent("/", "/proc", "rw,nosuid,nodev,noexec,relatime", "proc", "proc", "rw"),
				// Place(pathWantMnt, want.Bytes())
				ent(ignore, pathWantMnt, "ro,nosuid,nodev,relatime", "tmpfs", "rootfs", ignore),
			)
			want := new(bytes.Buffer)
			if err := gob.NewEncoder(want).Encode(mnt); err != nil {
				_, _ = output.WriteTo(os.Stdout)
				t.Fatalf("cannot serialise expected mount points: %v", err)
			}
			c.Place(pathWantMnt, want.Bytes())

			if tc.ro {
				c.Remount("/", syscall.MS_RDONLY)
			}

			if err := c.Start(); err != nil {
				_, _ = output.WriteTo(os.Stdout)
				hlog.PrintBaseError(err, "start:")
				t.Fatalf("cannot start container: %v", err)
			} else if err = c.Serve(); err != nil {
				_, _ = output.WriteTo(os.Stdout)
				hlog.PrintBaseError(err, "serve:")
				t.Errorf("cannot serve setup params: %v", err)
			}
			if err := c.Wait(); err != nil {
				_, _ = output.WriteTo(os.Stdout)
				hlog.PrintBaseError(err, "wait:")
				t.Fatalf("wait: %v", err)
			}
		})
	}
}

func ent(root, target, vfsOptstr, fsType, source, fsOptstr string) *vfs.MountInfoEntry {
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

func hostnameFromTestCase(name string) string {
	return "test-" + strings.Join(strings.Fields(name), "-")
}

func testContainerCancel(
	containerExtra func(c *container.Container),
	waitCheck func(t *testing.T, c *container.Container),
) func(t *testing.T) {
	return func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), helperDefaultTimeout)

		c := helperNewContainer(ctx, "block")
		c.Stdout, c.Stderr = os.Stdout, os.Stderr
		c.WaitDelay = helperDefaultTimeout
		if containerExtra != nil {
			containerExtra(c)
		}

		ready := make(chan struct{})
		if r, w, err := os.Pipe(); err != nil {
			t.Fatalf("cannot pipe: %v", err)
		} else {
			c.ExtraFiles = append(c.ExtraFiles, w)
			go func() {
				defer close(ready)
				if _, err = r.Read(make([]byte, 1)); err != nil {
					panic(err.Error())
				}
			}()
		}

		if err := c.Start(); err != nil {
			hlog.PrintBaseError(err, "start:")
			t.Fatalf("cannot start container: %v", err)
		} else if err = c.Serve(); err != nil {
			hlog.PrintBaseError(err, "serve:")
			t.Errorf("cannot serve setup params: %v", err)
		}
		<-ready
		cancel()
		waitCheck(t, c)
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

const (
	blockExitCodeInterrupt = 2
)

func init() {
	helperCommands = append(helperCommands, func(c command.Command) {
		c.Command("block", command.UsageInternal, func(args []string) error {
			if _, err := os.NewFile(3, "sync").Write([]byte{0}); err != nil {
				return fmt.Errorf("write to sync pipe: %v", err)
			}
			{
				sig := make(chan os.Signal, 1)
				signal.Notify(sig, os.Interrupt)
				go func() { <-sig; os.Exit(blockExitCodeInterrupt) }()
			}
			select {}
		})

		c.Command("container", command.UsageInternal, func(args []string) error {
			if len(args) != 1 {
				return syscall.EINVAL
			}
			tc := containerTestCases[0]
			if i, err := strconv.Atoi(args[0]); err != nil {
				return fmt.Errorf("cannot parse test case index: %v", err)
			} else {
				tc = containerTestCases[i]
			}

			if uid := syscall.Getuid(); uid != tc.uid {
				return fmt.Errorf("uid: %d, want %d", uid, tc.uid)
			}
			if gid := syscall.Getgid(); gid != tc.gid {
				return fmt.Errorf("gid: %d, want %d", gid, tc.gid)
			}

			wantHost := hostnameFromTestCase(tc.name)
			if host, err := os.Hostname(); err != nil {
				return fmt.Errorf("cannot get hostname: %v", err)
			} else if host != wantHost {
				return fmt.Errorf("hostname: %q, want %q", host, wantHost)
			}

			if p, err := os.ReadFile("/etc/hostname"); err != nil {
				return fmt.Errorf("cannot read /etc/hostname: %v", err)
			} else if string(p) != wantHost {
				return fmt.Errorf("/etc/hostname: %q, want %q", string(p), wantHost)
			}

			if _, err := os.Create(pathReadonly + "/nonexistent"); !errors.Is(err, syscall.EROFS) {
				return err
			}

			{
				var fail bool

				var mnt []*vfs.MountInfoEntry
				if f, err := os.Open(pathWantMnt); err != nil {
					return fmt.Errorf("cannot open expected mount points: %v", err)
				} else if err = gob.NewDecoder(f).Decode(&mnt); err != nil {
					return fmt.Errorf("cannot parse expected mount points: %v", err)
				} else if err = f.Close(); err != nil {
					return fmt.Errorf("cannot close expected mount points: %v", err)
				}

				if tc.ro && len(mnt) > 0 {
					// Remount("/", syscall.MS_RDONLY)
					mnt[0].VfsOptstr = "ro,nosuid,nodev"
				}

				var d *vfs.MountInfoDecoder
				if f, err := os.Open("/proc/self/mountinfo"); err != nil {
					return fmt.Errorf("cannot open mountinfo: %v", err)
				} else {
					d = vfs.NewMountInfoDecoder(f)
				}

				i := 0
				for cur := range d.Entries() {
					if i == len(mnt) {
						return fmt.Errorf("got more than %d entries", len(mnt))
					}

					// ugly hack but should be reliable and is less likely to false negative than comparing by parsed flags
					cur.VfsOptstr = strings.TrimSuffix(cur.VfsOptstr, ",relatime")
					cur.VfsOptstr = strings.TrimSuffix(cur.VfsOptstr, ",noatime")
					mnt[i].VfsOptstr = strings.TrimSuffix(mnt[i].VfsOptstr, ",relatime")
					mnt[i].VfsOptstr = strings.TrimSuffix(mnt[i].VfsOptstr, ",noatime")

					if !cur.EqualWithIgnore(mnt[i], "\x00") {
						fail = true
						log.Printf("[FAIL] %s", cur)
					} else {
						log.Printf("[ OK ] %s", cur)
					}

					i++
				}
				if err := d.Err(); err != nil {
					return fmt.Errorf("cannot parse mountinfo: %v", err)
				}

				if i != len(mnt) {
					return fmt.Errorf("got %d entries, want %d", i, len(mnt))
				}

				if fail {
					return errors.New("one or more mountinfo entries do not match")
				}
			}

			return nil
		})
	})
}
