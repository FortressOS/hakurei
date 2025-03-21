package vfs_test

import (
	"errors"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"git.gensokyo.uk/security/fortify/sandbox/vfs"
)

func TestParseMountInfo(t *testing.T) {
	testCases := []struct {
		name      string
		sample    string
		wantErr   error
		wantError string
		want      []*wantMountInfo
	}{
		{"count", sampleMountinfoShort + `
21 20 0:53/ /mnt/test rw,relatime - tmpfs  rw
21 16 0:17 / /sys/fs/cgroup rw,nosuid,nodev,noexec,relatime - tmpfs tmpfs rw,mode=755`, vfs.ErrMountInfoFields, "", nil},

		{"sep", sampleMountinfoShort + `
21 20 0:53 / /mnt/test rw,relatime shared:212 _ tmpfs  rw
21 16 0:17 / /sys/fs/cgroup rw,nosuid,nodev,noexec,relatime - tmpfs tmpfs rw,mode=755`, vfs.ErrMountInfoSep, "", nil},

		{"id", sampleMountinfoShort + `
id 20 0:53 / /mnt/test rw,relatime shared:212 - tmpfs  rw
21 16 0:17 / /sys/fs/cgroup rw,nosuid,nodev,noexec,relatime - tmpfs tmpfs rw,mode=755`, strconv.ErrSyntax, "", nil},

		{"parent", sampleMountinfoShort + `
21 parent 0:53 / /mnt/test rw,relatime shared:212 - tmpfs  rw
21 16 0:17 / /sys/fs/cgroup rw,nosuid,nodev,noexec,relatime - tmpfs tmpfs rw,mode=755`, strconv.ErrSyntax, "", nil},

		{"devno", sampleMountinfoShort + `
21 20 053 / /mnt/test rw,relatime shared:212 - tmpfs  rw
21 16 0:17 / /sys/fs/cgroup rw,nosuid,nodev,noexec,relatime - tmpfs tmpfs rw,mode=755`, nil, "unexpected EOF", nil},

		{"maj", sampleMountinfoShort + `
21 20 maj:53 / /mnt/test rw,relatime shared:212 - tmpfs  rw
21 16 0:17 / /sys/fs/cgroup rw,nosuid,nodev,noexec,relatime - tmpfs tmpfs rw,mode=755`, nil, "expected integer", nil},

		{"min", sampleMountinfoShort + `
21 20 0:min / /mnt/test rw,relatime shared:212 - tmpfs  rw
21 16 0:17 / /sys/fs/cgroup rw,nosuid,nodev,noexec,relatime - tmpfs tmpfs rw,mode=755`, nil, "expected integer", nil},

		{"mountroot", sampleMountinfoShort + `
21 20 0:53  /mnt/test rw,relatime - tmpfs  rw
21 16 0:17 / /sys/fs/cgroup rw,nosuid,nodev,noexec,relatime - tmpfs tmpfs rw,mode=755`, vfs.ErrMountInfoEmpty, "", nil},

		{"target", sampleMountinfoShort + `
21 20 0:53 /  rw,relatime - tmpfs  rw
21 16 0:17 / /sys/fs/cgroup rw,nosuid,nodev,noexec,relatime - tmpfs tmpfs rw,mode=755`, vfs.ErrMountInfoEmpty, "", nil},

		{"vfs options", sampleMountinfoShort + `
21 20 0:53 / /mnt/test  - tmpfs  rw
21 16 0:17 / /sys/fs/cgroup rw,nosuid,nodev,noexec,relatime - tmpfs tmpfs rw,mode=755`, vfs.ErrMountInfoEmpty, "", nil},

		{"FS type", sampleMountinfoShort + `
21 20 0:53 / /mnt/test rw,relatime -   rw
21 16 0:17 / /sys/fs/cgroup rw,nosuid,nodev,noexec,relatime - tmpfs tmpfs rw,mode=755`, vfs.ErrMountInfoEmpty, "", nil},

		{"base", sampleMountinfoShort, nil, "", []*wantMountInfo{
			m(15, 20, 0, 3, "/", "/proc", "rw,relatime", o(), "proc", "/proc", "rw", syscall.MS_RELATIME, nil),
			m(16, 20, 0, 15, "/", "/sys", "rw,relatime", o(), "sysfs", "/sys", "rw", syscall.MS_RELATIME, nil),
			m(17, 20, 0, 5, "/", "/dev", "rw,relatime", o(), "devtmpfs", "udev", "rw,size=1983516k,nr_inodes=495879,mode=755", syscall.MS_RELATIME, nil),
			m(18, 17, 0, 10, "/", "/dev/pts", "rw,relatime", o(), "devpts", "devpts", "rw,gid=5,mode=620,ptmxmode=000", syscall.MS_RELATIME, nil),
			m(19, 17, 0, 16, "/", "/dev/shm", "rw,relatime", o(), "tmpfs", "tmpfs", "rw", syscall.MS_RELATIME, nil),
			m(20, 1, 8, 4, "/", "/", "ro,noatime,nodiratime,meow", o(), "ext3", "/dev/sda4", "rw,errors=continue,user_xattr,acl,barrier=0,data=ordered", syscall.MS_RDONLY|syscall.MS_NOATIME|syscall.MS_NODIRATIME, []string{"meow"}),
		}},

		{"sample", sampleMountinfo, nil, "", []*wantMountInfo{
			m(15, 20, 0, 3, "/", "/proc", "rw,relatime", o(), "proc", "/proc", "rw", syscall.MS_RELATIME, nil),
			m(16, 20, 0, 15, "/", "/sys", "rw,relatime", o(), "sysfs", "/sys", "rw", syscall.MS_RELATIME, nil),
			m(17, 20, 0, 5, "/", "/dev", "rw,relatime", o(), "devtmpfs", "udev", "rw,size=1983516k,nr_inodes=495879,mode=755", syscall.MS_RELATIME, nil),
			m(18, 17, 0, 10, "/", "/dev/pts", "rw,relatime", o(), "devpts", "devpts", "rw,gid=5,mode=620,ptmxmode=000", syscall.MS_RELATIME, nil),
			m(19, 17, 0, 16, "/", "/dev/shm", "rw,relatime", o(), "tmpfs", "tmpfs", "rw", syscall.MS_RELATIME, nil),
			m(20, 1, 8, 4, "/", "/", "rw,noatime", o(), "ext3", "/dev/sda4", "rw,errors=continue,user_xattr,acl,barrier=0,data=ordered", syscall.MS_NOATIME, nil),
			m(21, 16, 0, 17, "/", "/sys/fs/cgroup", "rw,nosuid,nodev,noexec,relatime", o(), "tmpfs", "tmpfs", "rw,mode=755", syscall.MS_NOSUID|syscall.MS_NODEV|syscall.MS_NOEXEC|syscall.MS_RELATIME, nil),
			m(22, 21, 0, 18, "/", "/sys/fs/cgroup/systemd", "rw,nosuid,nodev,noexec,relatime", o(), "cgroup", "cgroup", "rw,release_agent=/lib/systemd/systemd-cgroups-agent,name=systemd", syscall.MS_NOSUID|syscall.MS_NODEV|syscall.MS_NOEXEC|syscall.MS_RELATIME, nil),
			m(23, 21, 0, 19, "/", "/sys/fs/cgroup/cpuset", "rw,nosuid,nodev,noexec,relatime", o(), "cgroup", "cgroup", "rw,cpuset", syscall.MS_NOSUID|syscall.MS_NODEV|syscall.MS_NOEXEC|syscall.MS_RELATIME, nil),
			m(24, 21, 0, 20, "/", "/sys/fs/cgroup/ns", "rw,nosuid,nodev,noexec,relatime", o(), "cgroup", "cgroup", "rw,ns", syscall.MS_NOSUID|syscall.MS_NODEV|syscall.MS_NOEXEC|syscall.MS_RELATIME, nil),
			m(25, 21, 0, 21, "/", "/sys/fs/cgroup/cpu", "rw,nosuid,nodev,noexec,relatime", o(), "cgroup", "cgroup", "rw,cpu", syscall.MS_NOSUID|syscall.MS_NODEV|syscall.MS_NOEXEC|syscall.MS_RELATIME, nil),
			m(26, 21, 0, 22, "/", "/sys/fs/cgroup/cpuacct", "rw,nosuid,nodev,noexec,relatime", o(), "cgroup", "cgroup", "rw,cpuacct", syscall.MS_NOSUID|syscall.MS_NODEV|syscall.MS_NOEXEC|syscall.MS_RELATIME, nil),
			m(27, 21, 0, 23, "/", "/sys/fs/cgroup/memory", "rw,nosuid,nodev,noexec,relatime", o(), "cgroup", "cgroup", "rw,memory", syscall.MS_NOSUID|syscall.MS_NODEV|syscall.MS_NOEXEC|syscall.MS_RELATIME, nil),
			m(28, 21, 0, 24, "/", "/sys/fs/cgroup/devices", "rw,nosuid,nodev,noexec,relatime", o(), "cgroup", "cgroup", "rw,devices", syscall.MS_NOSUID|syscall.MS_NODEV|syscall.MS_NOEXEC|syscall.MS_RELATIME, nil),
			m(29, 21, 0, 25, "/", "/sys/fs/cgroup/freezer", "rw,nosuid,nodev,noexec,relatime", o(), "cgroup", "cgroup", "rw,freezer", syscall.MS_NOSUID|syscall.MS_NODEV|syscall.MS_NOEXEC|syscall.MS_RELATIME, nil),
			m(30, 21, 0, 26, "/", "/sys/fs/cgroup/net_cls", "rw,nosuid,nodev,noexec,relatime", o(), "cgroup", "cgroup", "rw,net_cls", syscall.MS_NOSUID|syscall.MS_NODEV|syscall.MS_NOEXEC|syscall.MS_RELATIME, nil),
			m(31, 21, 0, 27, "/", "/sys/fs/cgroup/blkio", "rw,nosuid,nodev,noexec,relatime", o(), "cgroup", "cgroup", "rw,blkio", syscall.MS_NOSUID|syscall.MS_NODEV|syscall.MS_NOEXEC|syscall.MS_RELATIME, nil),
			m(32, 16, 0, 28, "/", "/sys/kernel/security", "rw,relatime", o(), "autofs", "systemd-1", "rw,fd=22,pgrp=1,timeout=300,minproto=5,maxproto=5,direct", syscall.MS_RELATIME, nil),
			m(33, 17, 0, 29, "/", "/dev/hugepages", "rw,relatime", o(), "autofs", "systemd-1", "rw,fd=23,pgrp=1,timeout=300,minproto=5,maxproto=5,direct", syscall.MS_RELATIME, nil),
			m(34, 16, 0, 30, "/", "/sys/kernel/debug", "rw,relatime", o(), "autofs", "systemd-1", "rw,fd=24,pgrp=1,timeout=300,minproto=5,maxproto=5,direct", syscall.MS_RELATIME, nil),
			m(35, 15, 0, 31, "/", "/proc/sys/fs/binfmt_misc", "rw,relatime", o(), "autofs", "systemd-1", "rw,fd=25,pgrp=1,timeout=300,minproto=5,maxproto=5,direct", syscall.MS_RELATIME, nil),
			m(36, 17, 0, 32, "/", "/dev/mqueue", "rw,relatime", o(), "autofs", "systemd-1", "rw,fd=26,pgrp=1,timeout=300,minproto=5,maxproto=5,direct", syscall.MS_RELATIME, nil),
			m(37, 15, 0, 14, "/", "/proc/bus/usb", "rw,relatime", o(), "usbfs", "/proc/bus/usb", "rw", syscall.MS_RELATIME, nil),
			m(38, 33, 0, 33, "/", "/dev/hugepages", "rw,relatime", o(), "hugetlbfs", "hugetlbfs", "rw", syscall.MS_RELATIME, nil),
			m(39, 36, 0, 12, "/", "/dev/mqueue", "rw,relatime", o(), "mqueue", "mqueue", "rw", syscall.MS_RELATIME, nil),
			m(40, 20, 8, 6, "/", "/boot", "rw,noatime", o(), "ext3", "/dev/sda6", "rw,errors=continue,barrier=0,data=ordered", syscall.MS_NOATIME, nil),
			m(41, 20, 253, 0, "/", "/home/kzak", "rw,noatime", o(), "ext4", "/dev/mapper/kzak-home", "rw,barrier=1,data=ordered", syscall.MS_NOATIME, nil),
			m(42, 35, 0, 34, "/", "/proc/sys/fs/binfmt_misc", "rw,relatime", o(), "binfmt_misc", "none", "rw", syscall.MS_RELATIME, nil),
			m(43, 16, 0, 35, "/", "/sys/fs/fuse/connections", "rw,relatime", o(), "fusectl", "fusectl", "rw", syscall.MS_RELATIME, nil),
			m(44, 41, 0, 36, "/", "/home/kzak/.gvfs", "rw,nosuid,nodev,relatime", o(), "fuse.gvfs-fuse-daemon", "gvfs-fuse-daemon", "rw,user_id=500,group_id=500", syscall.MS_NOSUID|syscall.MS_NODEV|syscall.MS_RELATIME, nil),
			m(45, 20, 0, 37, "/", "/var/lib/nfs/rpc_pipefs", "rw,relatime", o(), "rpc_pipefs", "sunrpc", "rw", syscall.MS_RELATIME, nil),
			m(47, 20, 0, 38, "/", "/mnt/sounds", "rw,relatime", o(), "cifs", "//foo.home/bar/", "rw,unc=\\\\foo.home\\bar,username=kzak,domain=SRGROUP,uid=0,noforceuid,gid=0,noforcegid,addr=192.168.111.1,posixpaths,serverino,acl,rsize=16384,wsize=57344", syscall.MS_RELATIME, nil),
			m(49, 20, 0, 56, "/", "/mnt/test/foobar", "rw,relatime", o("shared:323"), "tmpfs", "tmpfs", "rw", syscall.MS_RELATIME, nil),
		}},

		{"sample nosrc", sampleMountinfoNoSrc, nil, "", []*wantMountInfo{
			m(15, 20, 0, 3, "/", "/proc", "rw,relatime", o(), "proc", "/proc", "rw", syscall.MS_RELATIME, nil),
			m(16, 20, 0, 15, "/", "/sys", "rw,relatime", o(), "sysfs", "/sys", "rw", syscall.MS_RELATIME, nil),
			m(17, 20, 0, 5, "/", "/dev", "rw,relatime", o(), "devtmpfs", "udev", "rw,size=1983516k,nr_inodes=495879,mode=755", syscall.MS_RELATIME, nil),
			m(18, 17, 0, 10, "/", "/dev/pts", "rw,relatime", o(), "devpts", "devpts", "rw,gid=5,mode=620,ptmxmode=000", syscall.MS_RELATIME, nil),
			m(19, 17, 0, 16, "/", "/dev/shm", "rw,relatime", o(), "tmpfs", "tmpfs", "rw", syscall.MS_RELATIME, nil),
			m(20, 1, 8, 4, "/", "/", "rw,noatime", o(), "ext3", "/dev/sda4", "rw,errors=continue,user_xattr,acl,barrier=0,data=ordered", syscall.MS_NOATIME, nil),
			m(21, 20, 0, 53, "/", "/mnt/test", "rw,relatime", o("shared:212"), "tmpfs", "", "rw", syscall.MS_RELATIME, nil),
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, n, err := vfs.ParseMountInfo(strings.NewReader(tc.sample))
			if !errors.Is(err, tc.wantErr) {
				if tc.wantError == "" {
					t.Errorf("ParseMountInfo: error = %v, wantErr %v",
						err, tc.wantErr)
				} else if err != nil && err.Error() != tc.wantError {
					t.Errorf("ParseMountInfo: error = %q, wantError %q",
						err, tc.wantError)
				}
			}

			wantCount := len(tc.want)
			if tc.wantErr != nil || tc.wantError != "" {
				wantCount = -1
			}
			if n != wantCount {
				t.Errorf("ParseMountInfo: got %d entries, want %d", n, wantCount)
			}

			i := 0
			for cur := got; cur != nil; cur = cur.Next {
				if i == len(tc.want) {
					t.Errorf("ParseMountInfo: got more than %d entries", len(tc.want))
					break
				}

				if !reflect.DeepEqual(&cur.MountInfoEntry, &tc.want[i].MountInfoEntry) {
					t.Errorf("ParseMountInfo: entry %d\ngot:  %#v\nwant: %#v",
						i, cur.MountInfoEntry, tc.want[i])
				}

				flags, unmatched := cur.Flags()
				if flags != tc.want[i].flags {
					t.Errorf("Flags(%q): %#x, want %#x",
						cur.VfsOptstr, flags, tc.want[i].flags)
				}
				if !slices.Equal(unmatched, tc.want[i].unmatched) {
					t.Errorf("Flags(%q): unmatched = %#q, want %#q",
						cur.VfsOptstr, unmatched, tc.want[i].unmatched)
				}

				i++
			}

			if i != len(tc.want) {
				t.Errorf("ParseMountInfo: got %d entries, want %d", i, len(tc.want))
			}
		})
	}
}

type wantMountInfo struct {
	vfs.MountInfoEntry
	flags     uintptr
	unmatched []string
}

func m(
	id, parent, maj, min int, root, target, vfsOptstr string, optFields []string, fsType, source, fsOptstr string,
	flags uintptr, unmatched []string,
) *wantMountInfo {
	return &wantMountInfo{
		vfs.MountInfoEntry{
			ID:        id,
			Parent:    parent,
			Devno:     vfs.DevT{maj, min},
			Root:      root,
			Target:    target,
			VfsOptstr: vfsOptstr,
			OptFields: optFields,
			FsType:    fsType,
			Source:    source,
			FsOptstr:  fsOptstr,
		}, flags, unmatched,
	}
}

func o(field ...string) []string {
	if field == nil {
		return []string{}
	}
	return field
}

const (
	sampleMountinfoShort = `15 20 0:3 / /proc rw,relatime - proc /proc rw
16 20 0:15 / /sys rw,relatime - sysfs /sys rw
17 20 0:5 / /dev rw,relatime - devtmpfs udev rw,size=1983516k,nr_inodes=495879,mode=755
18 17 0:10 / /dev/pts rw,relatime - devpts devpts rw,gid=5,mode=620,ptmxmode=000
19 17 0:16 / /dev/shm rw,relatime - tmpfs tmpfs rw
20 1 8:4 / / ro,noatime,nodiratime,meow - ext3 /dev/sda4 rw,errors=continue,user_xattr,acl,barrier=0,data=ordered`

	sampleMountinfo = `15 20 0:3 / /proc rw,relatime - proc /proc rw
16 20 0:15 / /sys rw,relatime - sysfs /sys rw
17 20 0:5 / /dev rw,relatime - devtmpfs udev rw,size=1983516k,nr_inodes=495879,mode=755
18 17 0:10 / /dev/pts rw,relatime - devpts devpts rw,gid=5,mode=620,ptmxmode=000
19 17 0:16 / /dev/shm rw,relatime - tmpfs tmpfs rw
20 1 8:4 / / rw,noatime - ext3 /dev/sda4 rw,errors=continue,user_xattr,acl,barrier=0,data=ordered
21 16 0:17 / /sys/fs/cgroup rw,nosuid,nodev,noexec,relatime - tmpfs tmpfs rw,mode=755
22 21 0:18 / /sys/fs/cgroup/systemd rw,nosuid,nodev,noexec,relatime - cgroup cgroup rw,release_agent=/lib/systemd/systemd-cgroups-agent,name=systemd
23 21 0:19 / /sys/fs/cgroup/cpuset rw,nosuid,nodev,noexec,relatime - cgroup cgroup rw,cpuset
24 21 0:20 / /sys/fs/cgroup/ns rw,nosuid,nodev,noexec,relatime - cgroup cgroup rw,ns
25 21 0:21 / /sys/fs/cgroup/cpu rw,nosuid,nodev,noexec,relatime - cgroup cgroup rw,cpu
26 21 0:22 / /sys/fs/cgroup/cpuacct rw,nosuid,nodev,noexec,relatime - cgroup cgroup rw,cpuacct
27 21 0:23 / /sys/fs/cgroup/memory rw,nosuid,nodev,noexec,relatime - cgroup cgroup rw,memory
28 21 0:24 / /sys/fs/cgroup/devices rw,nosuid,nodev,noexec,relatime - cgroup cgroup rw,devices
29 21 0:25 / /sys/fs/cgroup/freezer rw,nosuid,nodev,noexec,relatime - cgroup cgroup rw,freezer
30 21 0:26 / /sys/fs/cgroup/net_cls rw,nosuid,nodev,noexec,relatime - cgroup cgroup rw,net_cls
31 21 0:27 / /sys/fs/cgroup/blkio rw,nosuid,nodev,noexec,relatime - cgroup cgroup rw,blkio
32 16 0:28 / /sys/kernel/security rw,relatime - autofs systemd-1 rw,fd=22,pgrp=1,timeout=300,minproto=5,maxproto=5,direct
33 17 0:29 / /dev/hugepages rw,relatime - autofs systemd-1 rw,fd=23,pgrp=1,timeout=300,minproto=5,maxproto=5,direct
34 16 0:30 / /sys/kernel/debug rw,relatime - autofs systemd-1 rw,fd=24,pgrp=1,timeout=300,minproto=5,maxproto=5,direct
35 15 0:31 / /proc/sys/fs/binfmt_misc rw,relatime - autofs systemd-1 rw,fd=25,pgrp=1,timeout=300,minproto=5,maxproto=5,direct
36 17 0:32 / /dev/mqueue rw,relatime - autofs systemd-1 rw,fd=26,pgrp=1,timeout=300,minproto=5,maxproto=5,direct
37 15 0:14 / /proc/bus/usb rw,relatime - usbfs /proc/bus/usb rw
38 33 0:33 / /dev/hugepages rw,relatime - hugetlbfs hugetlbfs rw
39 36 0:12 / /dev/mqueue rw,relatime - mqueue mqueue rw
40 20 8:6 / /boot rw,noatime - ext3 /dev/sda6 rw,errors=continue,barrier=0,data=ordered
41 20 253:0 / /home/kzak rw,noatime - ext4 /dev/mapper/kzak-home rw,barrier=1,data=ordered
42 35 0:34 / /proc/sys/fs/binfmt_misc rw,relatime - binfmt_misc none rw
43 16 0:35 / /sys/fs/fuse/connections rw,relatime - fusectl fusectl rw
44 41 0:36 / /home/kzak/.gvfs rw,nosuid,nodev,relatime - fuse.gvfs-fuse-daemon gvfs-fuse-daemon rw,user_id=500,group_id=500
45 20 0:37 / /var/lib/nfs/rpc_pipefs rw,relatime - rpc_pipefs sunrpc rw
47 20 0:38 / /mnt/sounds rw,relatime - cifs //foo.home/bar/ rw,unc=\\foo.home\bar,username=kzak,domain=SRGROUP,uid=0,noforceuid,gid=0,noforcegid,addr=192.168.111.1,posixpaths,serverino,acl,rsize=16384,wsize=57344
49 20 0:56 / /mnt/test/foobar rw,relatime shared:323 - tmpfs tmpfs rw`

	sampleMountinfoNoSrc = `15 20 0:3 / /proc rw,relatime - proc /proc rw
16 20 0:15 / /sys rw,relatime - sysfs /sys rw
17 20 0:5 / /dev rw,relatime - devtmpfs udev rw,size=1983516k,nr_inodes=495879,mode=755
18 17 0:10 / /dev/pts rw,relatime - devpts devpts rw,gid=5,mode=620,ptmxmode=000
19 17 0:16 / /dev/shm rw,relatime - tmpfs tmpfs rw
20 1 8:4 / / rw,noatime - ext3 /dev/sda4 rw,errors=continue,user_xattr,acl,barrier=0,data=ordered
21 20 0:53 / /mnt/test rw,relatime shared:212 - tmpfs  rw`
)
