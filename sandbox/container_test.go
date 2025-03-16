package sandbox_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/internal"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
	"git.gensokyo.uk/security/fortify/ldd"
	"git.gensokyo.uk/security/fortify/sandbox"
	"git.gensokyo.uk/security/fortify/sandbox/seccomp"
	check "git.gensokyo.uk/security/fortify/test/sandbox"
)

func TestContainer(t *testing.T) {
	{
		oldVerbose := fmsg.Load()
		oldOutput := sandbox.GetOutput()
		internal.InstallFmsg(true)
		t.Cleanup(func() { fmsg.Store(oldVerbose) })
		t.Cleanup(func() { sandbox.SetOutput(oldOutput) })
	}

	testCases := []struct {
		name  string
		flags sandbox.HardeningFlags
		ops   *sandbox.Ops
		mnt   []*check.Mntent
		host  string
	}{
		{"minimal", 0, new(sandbox.Ops), nil, "test-minimal"},
		{"allow", sandbox.FAllowUserns | sandbox.FAllowNet | sandbox.FAllowTTY,
			new(sandbox.Ops), nil, "test-minimal"},
		{"tmpfs", 0,
			new(sandbox.Ops).
				Tmpfs(fst.Tmp, 0, 0755),
			[]*check.Mntent{
				{FSName: "tmpfs", Dir: fst.Tmp, Type: "tmpfs", Opts: "\x00"},
			}, "test-tmpfs"},
		{"dev", sandbox.FAllowTTY, // go test output is not a tty
			new(sandbox.Ops).
				Dev("/dev"),
			[]*check.Mntent{
				{FSName: "devtmpfs", Dir: "/dev", Type: "tmpfs", Opts: "\x00"},
				{FSName: "devtmpfs", Dir: "/dev/null", Type: "devtmpfs", Opts: "\x00", Freq: -1, Passno: -1},
				{FSName: "devtmpfs", Dir: "/dev/zero", Type: "devtmpfs", Opts: "\x00", Freq: -1, Passno: -1},
				{FSName: "devtmpfs", Dir: "/dev/full", Type: "devtmpfs", Opts: "\x00", Freq: -1, Passno: -1},
				{FSName: "devtmpfs", Dir: "/dev/random", Type: "devtmpfs", Opts: "\x00", Freq: -1, Passno: -1},
				{FSName: "devtmpfs", Dir: "/dev/urandom", Type: "devtmpfs", Opts: "\x00", Freq: -1, Passno: -1},
				{FSName: "devtmpfs", Dir: "/dev/tty", Type: "devtmpfs", Opts: "\x00", Freq: -1, Passno: -1},
				{FSName: "devpts", Dir: "/dev/pts", Type: "devpts", Opts: "rw,nosuid,noexec,relatime,mode=620,ptmxmode=666", Freq: 0, Passno: 0},
			}, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			container := sandbox.New(ctx, os.Args[0], "-test.v",
				"-test.run=TestHelperCheckContainer", "--", "check", tc.host)
			container.Uid = 1000
			container.Gid = 100
			container.Hostname = tc.host
			container.CommandContext = commandContext
			container.Flags |= tc.flags
			container.Stdout, container.Stderr = os.Stdout, os.Stderr
			container.Ops = tc.ops
			if container.Args[5] == "" {
				if name, err := os.Hostname(); err != nil {
					t.Fatalf("cannot get hostname: %v", err)
				} else {
					container.Args[5] = name
				}
			}

			container.
				Tmpfs("/tmp", 0, 0755).
				Bind(os.Args[0], os.Args[0], 0)
			// in case test has cgo enabled
			var libPaths []string
			if entries, err := ldd.ExecFilter(ctx,
				commandContext,
				func(v []byte) []byte {
					return bytes.SplitN(v, []byte("TestHelperInit\n"), 2)[1]
				}, os.Args[0]); err != nil {
				log.Fatalf("ldd: %v", err)
			} else {
				libPaths = ldd.Path(entries)
			}
			for _, name := range libPaths {
				container.Bind(name, name, 0)
			}

			mnt := make([]*check.Mntent, 0, 3+len(libPaths))
			mnt = append(mnt, &check.Mntent{FSName: "rootfs", Dir: "/", Type: "tmpfs", Opts: "host_passthrough"})
			mnt = append(mnt, tc.mnt...)
			mnt = append(mnt,
				&check.Mntent{FSName: "tmpfs", Dir: "/tmp", Type: "tmpfs", Opts: "host_passthrough"},
				&check.Mntent{FSName: "\x00", Dir: os.Args[0], Type: "\x00", Opts: "\x00"})
			for _, name := range libPaths {
				mnt = append(mnt, &check.Mntent{FSName: "\x00", Dir: name, Type: "\x00", Opts: "\x00", Freq: -1, Passno: -1})
			}
			mnt = append(mnt, &check.Mntent{FSName: "proc", Dir: "/proc", Type: "proc", Opts: "rw,nosuid,nodev,noexec,relatime"})
			mntentWant := new(bytes.Buffer)
			if err := json.NewEncoder(mntentWant).Encode(mnt); err != nil {
				t.Fatalf("cannot serialise mntent: %v", err)
			}
			container.Stdin = mntentWant

			// needs /proc to check mntent
			container.Proc("/proc")

			if err := container.Start(); err != nil {
				fmsg.PrintBaseError(err, "start:")
				t.Fatalf("cannot start container: %v", err)
			} else if err = container.Serve(); err != nil {
				fmsg.PrintBaseError(err, "serve:")
				t.Errorf("cannot serve setup params: %v", err)
			}
			if err := container.Wait(); err != nil {
				fmsg.PrintBaseError(err, "wait:")
				t.Fatalf("wait: %v", err)
			}
		})
	}
}

func TestContainerString(t *testing.T) {
	container := sandbox.New(context.TODO(), "ldd", "/usr/bin/env")
	container.Flags |= sandbox.FAllowDevel
	container.Seccomp |= seccomp.FlagMultiarch
	want := `argv: ["ldd" "/usr/bin/env"], flags: 0x2, seccomp: 0x2e`
	if got := container.String(); got != want {
		t.Errorf("String: %s, want %s", got, want)
	}
}

func TestHelperInit(t *testing.T) {
	if len(os.Args) != 5 || os.Args[4] != "init" {
		return
	}
	sandbox.SetOutput(fmsg.Output{})
	sandbox.Init(fmsg.Prepare, internal.InstallFmsg)
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
	})
	t.Run("seccomp", func(t *testing.T) { check.MustAssertSeccomp() })
	t.Run("mntent", func(t *testing.T) { check.MustAssertMounts("", "/proc/mounts", "/proc/self/fd/0") })
}

func commandContext(ctx context.Context) *exec.Cmd {
	return exec.CommandContext(ctx, os.Args[0], "-test.v",
		"-test.run=TestHelperInit", "--", "init")
}
