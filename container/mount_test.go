package container

import (
	"os"
	"syscall"
	"testing"

	"hakurei.app/container/stub"
	"hakurei.app/container/vfs"
)

func TestBindMount(t *testing.T) {
	t.Parallel()

	checkSimple(t, "bindMount", []simpleTestCase{
		{"mount", func(k *kstub) error {
			return newProcPaths(k, hostPath).bindMount(nil, "/host/nix", "/sysroot/nix", syscall.MS_RDONLY)
		}, stub.Expect{Calls: []stub.Call{
			call("mount", stub.ExpectArgs{"/host/nix", "/sysroot/nix", "", uintptr(0x9000), ""}, nil, stub.UniqueError(0xbad)),
		}}, stub.UniqueError(0xbad)},

		{"success ne", func(k *kstub) error {
			return newProcPaths(k, hostPath).bindMount(k, "/host/nix", "/sysroot/.host-nix", syscall.MS_RDONLY)
		}, stub.Expect{Calls: []stub.Call{
			call("mount", stub.ExpectArgs{"/host/nix", "/sysroot/.host-nix", "", uintptr(0x9000), ""}, nil, nil),
			call("remount", stub.ExpectArgs{"/sysroot/.host-nix", uintptr(1)}, nil, nil),
		}}, nil},

		{"success", func(k *kstub) error {
			return newProcPaths(k, hostPath).bindMount(k, "/host/nix", "/sysroot/nix", syscall.MS_RDONLY)
		}, stub.Expect{Calls: []stub.Call{
			call("mount", stub.ExpectArgs{"/host/nix", "/sysroot/nix", "", uintptr(0x9000), ""}, nil, nil),
			call("remount", stub.ExpectArgs{"/sysroot/nix", uintptr(1)}, nil, nil),
		}}, nil},
	})
}

func TestRemount(t *testing.T) {
	t.Parallel()

	const sampleMountinfoNix = `254 407 253:0 / /host rw,relatime master:1 - ext4 /dev/disk/by-label/nixos rw
255 254 0:28 / /host/mnt/.ro-cwd ro,noatime master:2 - 9p cwd ro,access=client,msize=16384,trans=virtio
256 254 0:29 / /host/nix/.ro-store rw,relatime master:3 - 9p nix-store rw,cache=f,access=client,msize=16384,trans=virtio
257 254 0:30 / /host/nix/store rw,relatime master:4 - overlay overlay rw,lowerdir=/mnt-root/nix/.ro-store,upperdir=/mnt-root/nix/.rw-store/upper,workdir=/mnt-root/nix/.rw-store/work
258 257 0:30 / /host/nix/store ro,relatime master:5 - overlay overlay rw,lowerdir=/mnt-root/nix/.ro-store,upperdir=/mnt-root/nix/.rw-store/upper,workdir=/mnt-root/nix/.rw-store/work
259 254 0:33 / /host/tmp/shared rw,relatime master:6 - 9p shared rw,access=client,msize=16384,trans=virtio
260 254 0:34 / /host/tmp/xchg rw,relatime master:7 - 9p xchg rw,access=client,msize=16384,trans=virtio
261 254 0:22 / /host/proc rw,nosuid,nodev,noexec,relatime master:8 - proc proc rw
262 254 0:25 / /host/sys rw,nosuid,nodev,noexec,relatime master:9 - sysfs sysfs rw
263 262 0:7 / /host/sys/kernel/security rw,nosuid,nodev,noexec,relatime master:10 - securityfs securityfs rw
264 262 0:35 /../../.. /host/sys/fs/cgroup rw,nosuid,nodev,noexec,relatime master:11 - cgroup2 cgroup2 rw,nsdelegate,memory_recursiveprot
265 262 0:36 / /host/sys/fs/pstore rw,nosuid,nodev,noexec,relatime master:12 - pstore pstore rw
266 262 0:37 / /host/sys/fs/bpf rw,nosuid,nodev,noexec,relatime master:13 - bpf bpf rw,mode=700
267 262 0:12 / /host/sys/kernel/tracing rw,nosuid,nodev,noexec,relatime master:20 - tracefs tracefs rw
268 262 0:8 / /host/sys/kernel/debug rw,nosuid,nodev,noexec,relatime master:21 - debugfs debugfs rw
269 262 0:44 / /host/sys/kernel/config rw,nosuid,nodev,noexec,relatime master:64 - configfs configfs rw
270 262 0:45 / /host/sys/fs/fuse/connections rw,nosuid,nodev,noexec,relatime master:66 - fusectl fusectl rw
271 254 0:6 / /host/dev rw,nosuid master:14 - devtmpfs devtmpfs rw,size=200532k,nr_inodes=498943,mode=755
324 271 0:20 / /host/dev/pts rw,nosuid,noexec,relatime master:15 - devpts devpts rw,gid=3,mode=620,ptmxmode=666
378 271 0:21 / /host/dev/shm rw,nosuid,nodev master:16 - tmpfs tmpfs rw
379 271 0:19 / /host/dev/mqueue rw,nosuid,nodev,noexec,relatime master:19 - mqueue mqueue rw
388 271 0:38 / /host/dev/hugepages rw,nosuid,nodev,relatime master:22 - hugetlbfs hugetlbfs rw,pagesize=2M
397 254 0:23 / /host/run rw,nosuid,nodev master:17 - tmpfs tmpfs rw,size=1002656k,mode=755
398 397 0:24 / /host/run/keys rw,nosuid,nodev,relatime master:18 - ramfs ramfs rw,mode=750
399 397 0:39 / /host/run/credentials/systemd-journald.service ro,nosuid,nodev,noexec,relatime,nosymfollow master:23 - tmpfs tmpfs rw,size=1024k,nr_inodes=1024,mode=700,noswap
400 397 0:43 / /host/run/wrappers rw,nodev,relatime master:93 - tmpfs tmpfs rw,mode=755
401 397 0:61 / /host/run/credentials/getty@tty1.service ro,nosuid,nodev,noexec,relatime,nosymfollow master:240 - tmpfs tmpfs rw,size=1024k,nr_inodes=1024,mode=700,noswap
402 397 0:62 / /host/run/credentials/serial-getty@ttyS0.service ro,nosuid,nodev,noexec,relatime,nosymfollow master:288 - tmpfs tmpfs rw,size=1024k,nr_inodes=1024,mode=700,noswap
403 397 0:63 / /host/run/user/1000 rw,nosuid,nodev,relatime master:295 - tmpfs tmpfs rw,size=401060k,nr_inodes=100265,mode=700,uid=1000,gid=100
404 254 0:46 / /host/mnt/cwd rw,relatime master:96 - overlay overlay rw,lowerdir=/mnt/.ro-cwd,upperdir=/tmp/.cwd/upper,workdir=/tmp/.cwd/work
405 254 0:47 / /host/mnt/src rw,relatime master:99 - overlay overlay rw,lowerdir=/nix/store/ihcrl3zwvp2002xyylri2wz0drwajx4z-ns0pa7q2b1jpx9pbf1l9352x6rniwxjn-source,upperdir=/tmp/.src/upper,workdir=/tmp/.src/work
407 253 0:65 / / rw,nosuid,nodev,relatime - tmpfs rootfs rw,uid=10000,gid=10000
408 407 0:65 /sysroot /sysroot rw,nosuid,nodev,relatime - tmpfs rootfs rw,uid=10000,gid=10000
409 408 253:0 /bin /sysroot/bin rw,nosuid,nodev,relatime master:1 - ext4 /dev/disk/by-label/nixos rw
410 408 253:0 /home /sysroot/home rw,nosuid,nodev,relatime master:1 - ext4 /dev/disk/by-label/nixos rw
411 408 253:0 /lib64 /sysroot/lib64 rw,nosuid,nodev,relatime master:1 - ext4 /dev/disk/by-label/nixos rw
412 408 253:0 /lost+found /sysroot/lost+found rw,nosuid,nodev,relatime master:1 - ext4 /dev/disk/by-label/nixos rw
413 408 253:0 /nix /sysroot/nix rw,relatime master:1 - ext4 /dev/disk/by-label/nixos rw
414 413 0:29 / /sysroot/nix/.ro-store rw,relatime master:3 - 9p nix-store rw,cache=f,access=client,msize=16384,trans=virtio
415 413 0:30 / /sysroot/nix/store rw,relatime master:4 - overlay overlay rw,lowerdir=/mnt-root/nix/.ro-store,upperdir=/mnt-root/nix/.rw-store/upper,workdir=/mnt-root/nix/.rw-store/work
416 415 0:30 / /sysroot/nix/store ro,relatime master:5 - overlay overlay rw,lowerdir=/mnt-root/nix/.ro-store,upperdir=/mnt-root/nix/.rw-store/upper,workdir=/mnt-root/nix/.rw-store/work`

	checkSimple(t, "remount", []simpleTestCase{
		{"evalSymlinks", func(k *kstub) error {
			return newProcPaths(k, hostPath).remount(nil, "/sysroot/nix", syscall.MS_REC|syscall.MS_RDONLY|syscall.MS_NODEV)
		}, stub.Expect{Calls: []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/sysroot/nix"}, "/sysroot/nix", stub.UniqueError(6)),
		}}, stub.UniqueError(6)},

		{"open", func(k *kstub) error {
			return newProcPaths(k, hostPath).remount(nil, "/sysroot/nix", syscall.MS_REC|syscall.MS_RDONLY|syscall.MS_NODEV)
		}, stub.Expect{Calls: []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/sysroot/nix"}, "/sysroot/nix", nil),
			call("open", stub.ExpectArgs{"/sysroot/nix", 0x280000, uint32(0)}, 0xdead, stub.UniqueError(5)),
		}}, &os.PathError{Op: "open", Path: "/sysroot/nix", Err: stub.UniqueError(5)}},

		{"readlink", func(k *kstub) error {
			return newProcPaths(k, hostPath).remount(nil, "/sysroot/nix", syscall.MS_REC|syscall.MS_RDONLY|syscall.MS_NODEV)
		}, stub.Expect{Calls: []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/sysroot/nix"}, "/sysroot/nix", nil),
			call("open", stub.ExpectArgs{"/sysroot/nix", 0x280000, uint32(0)}, 0xdead, nil),
			call("readlink", stub.ExpectArgs{"/host/proc/self/fd/57005"}, "/sysroot/nix", stub.UniqueError(4)),
		}}, stub.UniqueError(4)},

		{"close", func(k *kstub) error {
			return newProcPaths(k, hostPath).remount(nil, "/sysroot/nix", syscall.MS_REC|syscall.MS_RDONLY|syscall.MS_NODEV)
		}, stub.Expect{Calls: []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/sysroot/nix"}, "/sysroot/nix", nil),
			call("open", stub.ExpectArgs{"/sysroot/nix", 0x280000, uint32(0)}, 0xdead, nil),
			call("readlink", stub.ExpectArgs{"/host/proc/self/fd/57005"}, "/sysroot/nix", nil),
			call("close", stub.ExpectArgs{0xdead}, nil, stub.UniqueError(3)),
		}}, &os.PathError{Op: "close", Path: "/sysroot/nix", Err: stub.UniqueError(3)}},

		{"mountinfo no match", func(k *kstub) error {
			return newProcPaths(k, hostPath).remount(k, "/sysroot/nix", syscall.MS_REC|syscall.MS_RDONLY|syscall.MS_NODEV)
		}, stub.Expect{Calls: []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/sysroot/nix"}, "/sysroot/.hakurei", nil),
			call("verbosef", stub.ExpectArgs{"target resolves to %q", []any{"/sysroot/.hakurei"}}, nil, nil),
			call("open", stub.ExpectArgs{"/sysroot/.hakurei", 0x280000, uint32(0)}, 0xdead, nil),
			call("readlink", stub.ExpectArgs{"/host/proc/self/fd/57005"}, "/sysroot/.hakurei", nil),
			call("close", stub.ExpectArgs{0xdead}, nil, nil),
			call("openNew", stub.ExpectArgs{"/host/proc/self/mountinfo"}, newConstFile(sampleMountinfoNix), nil),
		}}, &vfs.DecoderError{Op: "unfold", Line: -1, Err: vfs.UnfoldTargetError("/sysroot/.hakurei")}},

		{"mountinfo", func(k *kstub) error {
			return newProcPaths(k, hostPath).remount(nil, "/sysroot/nix", syscall.MS_REC|syscall.MS_RDONLY|syscall.MS_NODEV)
		}, stub.Expect{Calls: []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/sysroot/nix"}, "/sysroot/nix", nil),
			call("open", stub.ExpectArgs{"/sysroot/nix", 0x280000, uint32(0)}, 0xdead, nil),
			call("readlink", stub.ExpectArgs{"/host/proc/self/fd/57005"}, "/sysroot/nix", nil),
			call("close", stub.ExpectArgs{0xdead}, nil, nil),
			call("openNew", stub.ExpectArgs{"/host/proc/self/mountinfo"}, newConstFile("\x00"), nil),
		}}, &vfs.DecoderError{Op: "parse", Line: 0, Err: vfs.ErrMountInfoFields}},

		{"mount", func(k *kstub) error {
			return newProcPaths(k, hostPath).remount(nil, "/sysroot/nix", syscall.MS_REC|syscall.MS_RDONLY|syscall.MS_NODEV)
		}, stub.Expect{Calls: []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/sysroot/nix"}, "/sysroot/nix", nil),
			call("open", stub.ExpectArgs{"/sysroot/nix", 0x280000, uint32(0)}, 0xdead, nil),
			call("readlink", stub.ExpectArgs{"/host/proc/self/fd/57005"}, "/sysroot/nix", nil),
			call("close", stub.ExpectArgs{0xdead}, nil, nil),
			call("openNew", stub.ExpectArgs{"/host/proc/self/mountinfo"}, newConstFile(sampleMountinfoNix), nil),
			call("mount", stub.ExpectArgs{"none", "/sysroot/nix", "", uintptr(0x209027), ""}, nil, stub.UniqueError(2)),
		}}, stub.UniqueError(2)},

		{"mount propagate", func(k *kstub) error {
			return newProcPaths(k, hostPath).remount(nil, "/sysroot/nix", syscall.MS_REC|syscall.MS_RDONLY|syscall.MS_NODEV)
		}, stub.Expect{Calls: []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/sysroot/nix"}, "/sysroot/nix", nil),
			call("open", stub.ExpectArgs{"/sysroot/nix", 0x280000, uint32(0)}, 0xdead, nil),
			call("readlink", stub.ExpectArgs{"/host/proc/self/fd/57005"}, "/sysroot/nix", nil),
			call("close", stub.ExpectArgs{0xdead}, nil, nil),
			call("openNew", stub.ExpectArgs{"/host/proc/self/mountinfo"}, newConstFile(sampleMountinfoNix), nil),
			call("mount", stub.ExpectArgs{"none", "/sysroot/nix", "", uintptr(0x209027), ""}, nil, nil),
			call("mount", stub.ExpectArgs{"none", "/sysroot/nix/.ro-store", "", uintptr(0x209027), ""}, nil, stub.UniqueError(1)),
		}}, stub.UniqueError(1)},

		{"success toplevel", func(k *kstub) error {
			return newProcPaths(k, hostPath).remount(nil, "/sysroot/bin", syscall.MS_REC|syscall.MS_RDONLY|syscall.MS_NODEV)
		}, stub.Expect{Calls: []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/sysroot/bin"}, "/sysroot/bin", nil),
			call("open", stub.ExpectArgs{"/sysroot/bin", 0x280000, uint32(0)}, 0xbabe, nil),
			call("readlink", stub.ExpectArgs{"/host/proc/self/fd/47806"}, "/sysroot/bin", nil),
			call("close", stub.ExpectArgs{0xbabe}, nil, nil),
			call("openNew", stub.ExpectArgs{"/host/proc/self/mountinfo"}, newConstFile(sampleMountinfoNix), nil),
			call("mount", stub.ExpectArgs{"none", "/sysroot/bin", "", uintptr(0x209027), ""}, nil, nil),
		}}, nil},

		{"success EACCES", func(k *kstub) error {
			return newProcPaths(k, hostPath).remount(nil, "/sysroot/nix", syscall.MS_REC|syscall.MS_RDONLY|syscall.MS_NODEV)
		}, stub.Expect{Calls: []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/sysroot/nix"}, "/sysroot/nix", nil),
			call("open", stub.ExpectArgs{"/sysroot/nix", 0x280000, uint32(0)}, 0xdead, nil),
			call("readlink", stub.ExpectArgs{"/host/proc/self/fd/57005"}, "/sysroot/nix", nil),
			call("close", stub.ExpectArgs{0xdead}, nil, nil),
			call("openNew", stub.ExpectArgs{"/host/proc/self/mountinfo"}, newConstFile(sampleMountinfoNix), nil),
			call("mount", stub.ExpectArgs{"none", "/sysroot/nix", "", uintptr(0x209027), ""}, nil, nil),
			call("mount", stub.ExpectArgs{"none", "/sysroot/nix/.ro-store", "", uintptr(0x209027), ""}, nil, syscall.EACCES),
			call("mount", stub.ExpectArgs{"none", "/sysroot/nix/store", "", uintptr(0x209027), ""}, nil, nil),
		}}, nil},

		{"success no propagate", func(k *kstub) error {
			return newProcPaths(k, hostPath).remount(nil, "/sysroot/nix", syscall.MS_RDONLY|syscall.MS_NODEV)
		}, stub.Expect{Calls: []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/sysroot/nix"}, "/sysroot/nix", nil),
			call("open", stub.ExpectArgs{"/sysroot/nix", 0x280000, uint32(0)}, 0xdead, nil),
			call("readlink", stub.ExpectArgs{"/host/proc/self/fd/57005"}, "/sysroot/nix", nil),
			call("close", stub.ExpectArgs{0xdead}, nil, nil),
			call("openNew", stub.ExpectArgs{"/host/proc/self/mountinfo"}, newConstFile(sampleMountinfoNix), nil),
			call("mount", stub.ExpectArgs{"none", "/sysroot/nix", "", uintptr(0x209027), ""}, nil, nil),
		}}, nil},

		{"success case sensitive", func(k *kstub) error {
			return newProcPaths(k, hostPath).remount(nil, "/sysroot/nix", syscall.MS_REC|syscall.MS_RDONLY|syscall.MS_NODEV)
		}, stub.Expect{Calls: []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/sysroot/nix"}, "/sysroot/nix", nil),
			call("open", stub.ExpectArgs{"/sysroot/nix", 0x280000, uint32(0)}, 0xdead, nil),
			call("readlink", stub.ExpectArgs{"/host/proc/self/fd/57005"}, "/sysroot/nix", nil),
			call("close", stub.ExpectArgs{0xdead}, nil, nil),
			call("openNew", stub.ExpectArgs{"/host/proc/self/mountinfo"}, newConstFile(sampleMountinfoNix), nil),
			call("mount", stub.ExpectArgs{"none", "/sysroot/nix", "", uintptr(0x209027), ""}, nil, nil),
			call("mount", stub.ExpectArgs{"none", "/sysroot/nix/.ro-store", "", uintptr(0x209027), ""}, nil, nil),
			call("mount", stub.ExpectArgs{"none", "/sysroot/nix/store", "", uintptr(0x209027), ""}, nil, nil),
		}}, nil},

		{"success", func(k *kstub) error {
			return newProcPaths(k, hostPath).remount(k, "/sysroot/.nix", syscall.MS_REC|syscall.MS_RDONLY|syscall.MS_NODEV)
		}, stub.Expect{Calls: []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/sysroot/.nix"}, "/sysroot/NIX", nil),
			call("verbosef", stub.ExpectArgs{"target resolves to %q", []any{"/sysroot/NIX"}}, nil, nil),
			call("open", stub.ExpectArgs{"/sysroot/NIX", 0x280000, uint32(0)}, 0xdead, nil),
			call("readlink", stub.ExpectArgs{"/host/proc/self/fd/57005"}, "/sysroot/nix", nil),
			call("close", stub.ExpectArgs{0xdead}, nil, nil),
			call("openNew", stub.ExpectArgs{"/host/proc/self/mountinfo"}, newConstFile(sampleMountinfoNix), nil),
			call("mount", stub.ExpectArgs{"none", "/sysroot/nix", "", uintptr(0x209027), ""}, nil, nil),
			call("mount", stub.ExpectArgs{"none", "/sysroot/nix/.ro-store", "", uintptr(0x209027), ""}, nil, nil),
			call("mount", stub.ExpectArgs{"none", "/sysroot/nix/store", "", uintptr(0x209027), ""}, nil, nil),
		}}, nil},
	})
}

func TestRemountWithFlags(t *testing.T) {
	t.Parallel()

	checkSimple(t, "remountWithFlags", []simpleTestCase{
		{"noop unmatched", func(k *kstub) error {
			return remountWithFlags(k, k, &vfs.MountInfoNode{MountInfoEntry: &vfs.MountInfoEntry{VfsOptstr: "rw,relatime,cat"}}, 0)
		}, stub.Expect{Calls: []stub.Call{
			call("verbosef", stub.ExpectArgs{"unmatched vfs options: %q", []any{[]string{"cat"}}}, nil, nil),
		}}, nil},

		{"noop", func(k *kstub) error {
			return remountWithFlags(k, nil, &vfs.MountInfoNode{MountInfoEntry: &vfs.MountInfoEntry{VfsOptstr: "rw,relatime"}}, 0)
		}, stub.Expect{}, nil},

		{"success", func(k *kstub) error {
			return remountWithFlags(k, nil, &vfs.MountInfoNode{MountInfoEntry: &vfs.MountInfoEntry{VfsOptstr: "rw,relatime"}}, syscall.MS_RDONLY)
		}, stub.Expect{Calls: []stub.Call{
			call("mount", stub.ExpectArgs{"none", "", "", uintptr(0x209021), ""}, nil, nil),
		}}, nil},
	})
}

func TestMountTmpfs(t *testing.T) {
	t.Parallel()

	checkSimple(t, "mountTmpfs", []simpleTestCase{
		{"mkdirAll", func(k *kstub) error {
			return mountTmpfs(k, "ephemeral", "/sysroot/run/user/1000", 0, 1<<10, 0700)
		}, stub.Expect{Calls: []stub.Call{
			call("mkdirAll", stub.ExpectArgs{"/sysroot/run/user/1000", os.FileMode(0700)}, nil, stub.UniqueError(0)),
		}}, stub.UniqueError(0)},

		{"success no size", func(k *kstub) error {
			return mountTmpfs(k, "ephemeral", "/sysroot/run/user/1000", 0, 0, 0710)
		}, stub.Expect{Calls: []stub.Call{
			call("mkdirAll", stub.ExpectArgs{"/sysroot/run/user/1000", os.FileMode(0750)}, nil, nil),
			call("mount", stub.ExpectArgs{"ephemeral", "/sysroot/run/user/1000", "tmpfs", uintptr(0), "mode=0710"}, nil, nil),
		}}, nil},

		{"success", func(k *kstub) error {
			return mountTmpfs(k, "ephemeral", "/sysroot/run/user/1000", 0, 1<<10, 0700)
		}, stub.Expect{Calls: []stub.Call{
			call("mkdirAll", stub.ExpectArgs{"/sysroot/run/user/1000", os.FileMode(0700)}, nil, nil),
			call("mount", stub.ExpectArgs{"ephemeral", "/sysroot/run/user/1000", "tmpfs", uintptr(0), "mode=0700,size=1024"}, nil, nil),
		}}, nil},
	})
}

func TestParentPerm(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		perm os.FileMode
		want os.FileMode
	}{
		{0755, 0755},
		{0750, 0750},
		{0705, 0705},
		{0700, 0700},
		{050, 0750},
		{05, 0705},
		{0, 0700},
	}

	for _, tc := range testCases {
		t.Run(tc.perm.String(), func(t *testing.T) {
			t.Parallel()
			if got := parentPerm(tc.perm); got != tc.want {
				t.Errorf("parentPerm: %#o, want %#o", got, tc.want)
			}
		})
	}
}
