package container

import (
	"os"
	"syscall"
	"testing"

	"hakurei.app/container/vfs"
)

func TestBindMount(t *testing.T) {
	checkSimple(t, "bindMount", []simpleTestCase{
		{"mount", func(k syscallDispatcher) error {
			return newProcPaths(k, hostPath).bindMount("/host/nix", "/sysroot/nix", syscall.MS_RDONLY, true)
		}, [][]kexpect{{
			{"verbosef", expectArgs{"resolved %q flags %#x", []any{"/sysroot/nix", uintptr(1)}}, nil, nil},
			{"mount", expectArgs{"/host/nix", "/sysroot/nix", "", uintptr(0x9000), ""}, nil, errUnique},
		}}, wrapErrSuffix(errUnique, `cannot mount "/host/nix" on "/sysroot/nix":`)},

		{"success ne", func(k syscallDispatcher) error {
			return newProcPaths(k, hostPath).bindMount("/host/nix", "/sysroot/.host-nix", syscall.MS_RDONLY, false)
		}, [][]kexpect{{
			{"verbosef", expectArgs{"resolved %q on %q flags %#x", []any{"/host/nix", "/sysroot/.host-nix", uintptr(1)}}, nil, nil},
			{"mount", expectArgs{"/host/nix", "/sysroot/.host-nix", "", uintptr(0x9000), ""}, nil, nil},
			{"remount", expectArgs{"/sysroot/.host-nix", uintptr(1)}, nil, nil},
		}}, nil},

		{"success", func(k syscallDispatcher) error {
			return newProcPaths(k, hostPath).bindMount("/host/nix", "/sysroot/nix", syscall.MS_RDONLY, true)
		}, [][]kexpect{{
			{"verbosef", expectArgs{"resolved %q flags %#x", []any{"/sysroot/nix", uintptr(1)}}, nil, nil},
			{"mount", expectArgs{"/host/nix", "/sysroot/nix", "", uintptr(0x9000), ""}, nil, nil},
			{"remount", expectArgs{"/sysroot/nix", uintptr(1)}, nil, nil},
		}}, nil},
	})
}

func TestRemount(t *testing.T) {
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
407 253 0:65 / / rw,nosuid,nodev,relatime - tmpfs rootfs rw,uid=1000000,gid=1000000
408 407 0:65 /sysroot /sysroot rw,nosuid,nodev,relatime - tmpfs rootfs rw,uid=1000000,gid=1000000
409 408 253:0 /bin /sysroot/bin rw,nosuid,nodev,relatime master:1 - ext4 /dev/disk/by-label/nixos rw
410 408 253:0 /home /sysroot/home rw,nosuid,nodev,relatime master:1 - ext4 /dev/disk/by-label/nixos rw
411 408 253:0 /lib64 /sysroot/lib64 rw,nosuid,nodev,relatime master:1 - ext4 /dev/disk/by-label/nixos rw
412 408 253:0 /lost+found /sysroot/lost+found rw,nosuid,nodev,relatime master:1 - ext4 /dev/disk/by-label/nixos rw
413 408 253:0 /nix /sysroot/nix rw,relatime master:1 - ext4 /dev/disk/by-label/nixos rw
414 413 0:29 / /sysroot/nix/.ro-store rw,relatime master:3 - 9p nix-store rw,cache=f,access=client,msize=16384,trans=virtio
415 413 0:30 / /sysroot/nix/store rw,relatime master:4 - overlay overlay rw,lowerdir=/mnt-root/nix/.ro-store,upperdir=/mnt-root/nix/.rw-store/upper,workdir=/mnt-root/nix/.rw-store/work
416 415 0:30 / /sysroot/nix/store ro,relatime master:5 - overlay overlay rw,lowerdir=/mnt-root/nix/.ro-store,upperdir=/mnt-root/nix/.rw-store/upper,workdir=/mnt-root/nix/.rw-store/work`

	checkSimple(t, "remount", []simpleTestCase{
		{"evalSymlinks", func(k syscallDispatcher) error {
			return newProcPaths(k, hostPath).remount("/sysroot/nix", syscall.MS_REC|syscall.MS_RDONLY|syscall.MS_NODEV)
		}, [][]kexpect{{
			{"evalSymlinks", expectArgs{"/sysroot/nix"}, "/sysroot/nix", errUnique},
		}}, wrapErrSelf(errUnique)},

		{"open", func(k syscallDispatcher) error {
			return newProcPaths(k, hostPath).remount("/sysroot/nix", syscall.MS_REC|syscall.MS_RDONLY|syscall.MS_NODEV)
		}, [][]kexpect{{
			{"evalSymlinks", expectArgs{"/sysroot/nix"}, "/sysroot/nix", nil},
			{"open", expectArgs{"/sysroot/nix", 0x280000, uint32(0)}, 0xdeadbeef, errUnique},
		}}, wrapErrSuffix(errUnique, `cannot open "/sysroot/nix":`)},

		{"readlink", func(k syscallDispatcher) error {
			return newProcPaths(k, hostPath).remount("/sysroot/nix", syscall.MS_REC|syscall.MS_RDONLY|syscall.MS_NODEV)
		}, [][]kexpect{{
			{"evalSymlinks", expectArgs{"/sysroot/nix"}, "/sysroot/nix", nil},
			{"open", expectArgs{"/sysroot/nix", 0x280000, uint32(0)}, 0xdeadbeef, nil},
			{"readlink", expectArgs{"/host/proc/self/fd/3735928559"}, "/sysroot/nix", errUnique},
		}}, wrapErrSelf(errUnique)},

		{"close", func(k syscallDispatcher) error {
			return newProcPaths(k, hostPath).remount("/sysroot/nix", syscall.MS_REC|syscall.MS_RDONLY|syscall.MS_NODEV)
		}, [][]kexpect{{
			{"evalSymlinks", expectArgs{"/sysroot/nix"}, "/sysroot/nix", nil},
			{"open", expectArgs{"/sysroot/nix", 0x280000, uint32(0)}, 0xdeadbeef, nil},
			{"readlink", expectArgs{"/host/proc/self/fd/3735928559"}, "/sysroot/nix", nil},
			{"close", expectArgs{0xdeadbeef}, nil, errUnique},
		}}, wrapErrSuffix(errUnique, `cannot close "/sysroot/nix":`)},

		{"mountinfo stale", func(k syscallDispatcher) error {
			return newProcPaths(k, hostPath).remount("/sysroot/nix", syscall.MS_REC|syscall.MS_RDONLY|syscall.MS_NODEV)
		}, [][]kexpect{{
			{"evalSymlinks", expectArgs{"/sysroot/nix"}, "/sysroot/.hakurei", nil},
			{"verbosef", expectArgs{"target resolves to %q", []any{"/sysroot/.hakurei"}}, nil, nil},
			{"open", expectArgs{"/sysroot/.hakurei", 0x280000, uint32(0)}, 0xdeadbeef, nil},
			{"readlink", expectArgs{"/host/proc/self/fd/3735928559"}, "/sysroot/.hakurei", nil},
			{"close", expectArgs{0xdeadbeef}, nil, nil},
			{"openNew", expectArgs{"/host/proc/self/mountinfo"}, newConstFile(sampleMountinfoNix), nil},
		}}, msg.WrapErr(syscall.ESTALE, `mount point "/sysroot/.hakurei" never appeared in mountinfo`)},

		{"mountinfo", func(k syscallDispatcher) error {
			return newProcPaths(k, hostPath).remount("/sysroot/nix", syscall.MS_REC|syscall.MS_RDONLY|syscall.MS_NODEV)
		}, [][]kexpect{{
			{"evalSymlinks", expectArgs{"/sysroot/nix"}, "/sysroot/nix", nil},
			{"open", expectArgs{"/sysroot/nix", 0x280000, uint32(0)}, 0xdeadbeef, nil},
			{"readlink", expectArgs{"/host/proc/self/fd/3735928559"}, "/sysroot/nix", nil},
			{"close", expectArgs{0xdeadbeef}, nil, nil},
			{"openNew", expectArgs{"/host/proc/self/mountinfo"}, newConstFile("\x00"), nil},
		}}, wrapErrSuffix(vfs.ErrMountInfoFields, `cannot parse mountinfo:`)},

		{"mount", func(k syscallDispatcher) error {
			return newProcPaths(k, hostPath).remount("/sysroot/nix", syscall.MS_REC|syscall.MS_RDONLY|syscall.MS_NODEV)
		}, [][]kexpect{{
			{"evalSymlinks", expectArgs{"/sysroot/nix"}, "/sysroot/nix", nil},
			{"open", expectArgs{"/sysroot/nix", 0x280000, uint32(0)}, 0xdeadbeef, nil},
			{"readlink", expectArgs{"/host/proc/self/fd/3735928559"}, "/sysroot/nix", nil},
			{"close", expectArgs{0xdeadbeef}, nil, nil},
			{"openNew", expectArgs{"/host/proc/self/mountinfo"}, newConstFile(sampleMountinfoNix), nil},
			{"mount", expectArgs{"none", "/sysroot/nix", "", uintptr(0x209027), ""}, nil, errUnique},
		}}, wrapErrSuffix(errUnique, `cannot remount "/sysroot/nix":`)},

		{"mount propagate", func(k syscallDispatcher) error {
			return newProcPaths(k, hostPath).remount("/sysroot/nix", syscall.MS_REC|syscall.MS_RDONLY|syscall.MS_NODEV)
		}, [][]kexpect{{
			{"evalSymlinks", expectArgs{"/sysroot/nix"}, "/sysroot/nix", nil},
			{"open", expectArgs{"/sysroot/nix", 0x280000, uint32(0)}, 0xdeadbeef, nil},
			{"readlink", expectArgs{"/host/proc/self/fd/3735928559"}, "/sysroot/nix", nil},
			{"close", expectArgs{0xdeadbeef}, nil, nil},
			{"openNew", expectArgs{"/host/proc/self/mountinfo"}, newConstFile(sampleMountinfoNix), nil},
			{"mount", expectArgs{"none", "/sysroot/nix", "", uintptr(0x209027), ""}, nil, nil},
			{"mount", expectArgs{"none", "/sysroot/nix/.ro-store", "", uintptr(0x209027), ""}, nil, errUnique},
		}}, wrapErrSuffix(errUnique, `cannot propagate flags to "/sysroot/nix/.ro-store":`)},

		{"success toplevel", func(k syscallDispatcher) error {
			return newProcPaths(k, hostPath).remount("/sysroot/bin", syscall.MS_REC|syscall.MS_RDONLY|syscall.MS_NODEV)
		}, [][]kexpect{{
			{"evalSymlinks", expectArgs{"/sysroot/bin"}, "/sysroot/bin", nil},
			{"open", expectArgs{"/sysroot/bin", 0x280000, uint32(0)}, 0xbabe, nil},
			{"readlink", expectArgs{"/host/proc/self/fd/47806"}, "/sysroot/bin", nil},
			{"close", expectArgs{0xbabe}, nil, nil},
			{"openNew", expectArgs{"/host/proc/self/mountinfo"}, newConstFile(sampleMountinfoNix), nil},
			{"mount", expectArgs{"none", "/sysroot/bin", "", uintptr(0x209027), ""}, nil, nil},
		}}, nil},

		{"success EACCES", func(k syscallDispatcher) error {
			return newProcPaths(k, hostPath).remount("/sysroot/nix", syscall.MS_REC|syscall.MS_RDONLY|syscall.MS_NODEV)
		}, [][]kexpect{{
			{"evalSymlinks", expectArgs{"/sysroot/nix"}, "/sysroot/nix", nil},
			{"open", expectArgs{"/sysroot/nix", 0x280000, uint32(0)}, 0xdeadbeef, nil},
			{"readlink", expectArgs{"/host/proc/self/fd/3735928559"}, "/sysroot/nix", nil},
			{"close", expectArgs{0xdeadbeef}, nil, nil},
			{"openNew", expectArgs{"/host/proc/self/mountinfo"}, newConstFile(sampleMountinfoNix), nil},
			{"mount", expectArgs{"none", "/sysroot/nix", "", uintptr(0x209027), ""}, nil, nil},
			{"mount", expectArgs{"none", "/sysroot/nix/.ro-store", "", uintptr(0x209027), ""}, nil, syscall.EACCES},
			{"mount", expectArgs{"none", "/sysroot/nix/store", "", uintptr(0x209027), ""}, nil, nil},
		}}, nil},

		{"success no propagate", func(k syscallDispatcher) error {
			return newProcPaths(k, hostPath).remount("/sysroot/nix", syscall.MS_RDONLY|syscall.MS_NODEV)
		}, [][]kexpect{{
			{"evalSymlinks", expectArgs{"/sysroot/nix"}, "/sysroot/nix", nil},
			{"open", expectArgs{"/sysroot/nix", 0x280000, uint32(0)}, 0xdeadbeef, nil},
			{"readlink", expectArgs{"/host/proc/self/fd/3735928559"}, "/sysroot/nix", nil},
			{"close", expectArgs{0xdeadbeef}, nil, nil},
			{"openNew", expectArgs{"/host/proc/self/mountinfo"}, newConstFile(sampleMountinfoNix), nil},
			{"mount", expectArgs{"none", "/sysroot/nix", "", uintptr(0x209027), ""}, nil, nil},
		}}, nil},

		{"success case sensitive", func(k syscallDispatcher) error {
			return newProcPaths(k, hostPath).remount("/sysroot/nix", syscall.MS_REC|syscall.MS_RDONLY|syscall.MS_NODEV)
		}, [][]kexpect{{
			{"evalSymlinks", expectArgs{"/sysroot/nix"}, "/sysroot/nix", nil},
			{"open", expectArgs{"/sysroot/nix", 0x280000, uint32(0)}, 0xdeadbeef, nil},
			{"readlink", expectArgs{"/host/proc/self/fd/3735928559"}, "/sysroot/nix", nil},
			{"close", expectArgs{0xdeadbeef}, nil, nil},
			{"openNew", expectArgs{"/host/proc/self/mountinfo"}, newConstFile(sampleMountinfoNix), nil},
			{"mount", expectArgs{"none", "/sysroot/nix", "", uintptr(0x209027), ""}, nil, nil},
			{"mount", expectArgs{"none", "/sysroot/nix/.ro-store", "", uintptr(0x209027), ""}, nil, nil},
			{"mount", expectArgs{"none", "/sysroot/nix/store", "", uintptr(0x209027), ""}, nil, nil},
		}}, nil},

		{"success", func(k syscallDispatcher) error {
			return newProcPaths(k, hostPath).remount("/sysroot/.nix", syscall.MS_REC|syscall.MS_RDONLY|syscall.MS_NODEV)
		}, [][]kexpect{{
			{"evalSymlinks", expectArgs{"/sysroot/.nix"}, "/sysroot/NIX", nil},
			{"verbosef", expectArgs{"target resolves to %q", []any{"/sysroot/NIX"}}, nil, nil},
			{"open", expectArgs{"/sysroot/NIX", 0x280000, uint32(0)}, 0xdeadbeef, nil},
			{"readlink", expectArgs{"/host/proc/self/fd/3735928559"}, "/sysroot/nix", nil},
			{"close", expectArgs{0xdeadbeef}, nil, nil},
			{"openNew", expectArgs{"/host/proc/self/mountinfo"}, newConstFile(sampleMountinfoNix), nil},
			{"mount", expectArgs{"none", "/sysroot/nix", "", uintptr(0x209027), ""}, nil, nil},
			{"mount", expectArgs{"none", "/sysroot/nix/.ro-store", "", uintptr(0x209027), ""}, nil, nil},
			{"mount", expectArgs{"none", "/sysroot/nix/store", "", uintptr(0x209027), ""}, nil, nil},
		}}, nil},
	})
}

func TestRemountWithFlags(t *testing.T) {
	checkSimple(t, "remountWithFlags", []simpleTestCase{
		{"noop unmatched", func(k syscallDispatcher) error {
			return remountWithFlags(k, &vfs.MountInfoNode{MountInfoEntry: &vfs.MountInfoEntry{VfsOptstr: "rw,relatime,cat"}}, 0)
		}, [][]kexpect{{
			{"verbosef", expectArgs{"unmatched vfs options: %q", []any{[]string{"cat"}}}, nil, nil},
		}}, nil},

		{"noop", func(k syscallDispatcher) error {
			return remountWithFlags(k, &vfs.MountInfoNode{MountInfoEntry: &vfs.MountInfoEntry{VfsOptstr: "rw,relatime"}}, 0)
		}, nil, nil},

		{"success", func(k syscallDispatcher) error {
			return remountWithFlags(k, &vfs.MountInfoNode{MountInfoEntry: &vfs.MountInfoEntry{VfsOptstr: "rw,relatime"}}, syscall.MS_RDONLY)
		}, [][]kexpect{{
			{"mount", expectArgs{"none", "", "", uintptr(0x209021), ""}, nil, nil},
		}}, nil},
	})
}

func TestMountTmpfs(t *testing.T) {
	checkSimple(t, "mountTmpfs", []simpleTestCase{
		{"mkdirAll", func(k syscallDispatcher) error {
			return mountTmpfs(k, "ephemeral", "/sysroot/run/user/1000", 0, 1<<10, 0700)
		}, [][]kexpect{{
			{"mkdirAll", expectArgs{"/sysroot/run/user/1000", os.FileMode(0700)}, nil, errUnique},
		}}, wrapErrSelf(errUnique)},

		{"success no size", func(k syscallDispatcher) error {
			return mountTmpfs(k, "ephemeral", "/sysroot/run/user/1000", 0, 0, 0710)
		}, [][]kexpect{{
			{"mkdirAll", expectArgs{"/sysroot/run/user/1000", os.FileMode(0750)}, nil, nil},
			{"mount", expectArgs{"ephemeral", "/sysroot/run/user/1000", "tmpfs", uintptr(0), "mode=0710"}, nil, nil},
		}}, nil},

		{"success", func(k syscallDispatcher) error {
			return mountTmpfs(k, "ephemeral", "/sysroot/run/user/1000", 0, 1<<10, 0700)
		}, [][]kexpect{{
			{"mkdirAll", expectArgs{"/sysroot/run/user/1000", os.FileMode(0700)}, nil, nil},
			{"mount", expectArgs{"ephemeral", "/sysroot/run/user/1000", "tmpfs", uintptr(0), "mode=0700,size=1024"}, nil, nil},
		}}, nil},
	})
}

func TestParentPerm(t *testing.T) {
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
			if got := parentPerm(tc.perm); got != tc.want {
				t.Errorf("parentPerm: %#o, want %#o", got, tc.want)
			}
		})
	}
}

func TestEscapeOverlayDataSegment(t *testing.T) {
	testCases := []struct {
		name string
		s    string
		want string
	}{
		{"zero", zeroString, zeroString},
		{"multi", `\\\:,:,\\\`, `\\\\\\\:\,\:\,\\\\\\`},
		{"bwrap", `/path :,\`, `/path \:\,\\`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := EscapeOverlayDataSegment(tc.s); got != tc.want {
				t.Errorf("escapeOverlayDataSegment: %s, want %s", got, tc.want)
			}
		})
	}
}
