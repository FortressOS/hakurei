package vfs_test

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"git.gensokyo.uk/security/fortify/sandbox/vfs"
)

func TestParseMountInfo(t *testing.T) {
	testCases := []struct {
		name      string
		sample    string
		wantErr   error
		wantError string
		want      []vfs.MountInfoEntry
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

		{"base", sampleMountinfoShort, nil, "", []vfs.MountInfoEntry{
			{15, 20, dev(0, 3), "/", "/proc", "rw,relatime", opt(), "proc", "/proc", "rw"},
			{16, 20, dev(0, 15), "/", "/sys", "rw,relatime", opt(), "sysfs", "/sys", "rw"},
			{17, 20, dev(0, 5), "/", "/dev", "rw,relatime", opt(), "devtmpfs", "udev", "rw,size=1983516k,nr_inodes=495879,mode=755"},
			{18, 17, dev(0, 10), "/", "/dev/pts", "rw,relatime", opt(), "devpts", "devpts", "rw,gid=5,mode=620,ptmxmode=000"},
			{19, 17, dev(0, 16), "/", "/dev/shm", "rw,relatime", opt(), "tmpfs", "tmpfs", "rw"},
			{20, 1, dev(8, 4), "/", "/", "rw,noatime", opt(), "ext3", "/dev/sda4", "rw,errors=continue,user_xattr,acl,barrier=0,data=ordered"},
		}},

		{"sample", sampleMountinfo, nil, "", []vfs.MountInfoEntry{
			{15, 20, dev(0, 3), "/", "/proc", "rw,relatime", opt(), "proc", "/proc", "rw"},
			{16, 20, dev(0, 15), "/", "/sys", "rw,relatime", opt(), "sysfs", "/sys", "rw"},
			{17, 20, dev(0, 5), "/", "/dev", "rw,relatime", opt(), "devtmpfs", "udev", "rw,size=1983516k,nr_inodes=495879,mode=755"},
			{18, 17, dev(0, 10), "/", "/dev/pts", "rw,relatime", opt(), "devpts", "devpts", "rw,gid=5,mode=620,ptmxmode=000"},
			{19, 17, dev(0, 16), "/", "/dev/shm", "rw,relatime", opt(), "tmpfs", "tmpfs", "rw"},
			{20, 1, dev(8, 4), "/", "/", "rw,noatime", opt(), "ext3", "/dev/sda4", "rw,errors=continue,user_xattr,acl,barrier=0,data=ordered"},
			{21, 16, dev(0, 17), "/", "/sys/fs/cgroup", "rw,nosuid,nodev,noexec,relatime", opt(), "tmpfs", "tmpfs", "rw,mode=755"},
			{22, 21, dev(0, 18), "/", "/sys/fs/cgroup/systemd", "rw,nosuid,nodev,noexec,relatime", opt(), "cgroup", "cgroup", "rw,release_agent=/lib/systemd/systemd-cgroups-agent,name=systemd"},
			{23, 21, dev(0, 19), "/", "/sys/fs/cgroup/cpuset", "rw,nosuid,nodev,noexec,relatime", opt(), "cgroup", "cgroup", "rw,cpuset"},
			{24, 21, dev(0, 20), "/", "/sys/fs/cgroup/ns", "rw,nosuid,nodev,noexec,relatime", opt(), "cgroup", "cgroup", "rw,ns"},
			{25, 21, dev(0, 21), "/", "/sys/fs/cgroup/cpu", "rw,nosuid,nodev,noexec,relatime", opt(), "cgroup", "cgroup", "rw,cpu"},
			{26, 21, dev(0, 22), "/", "/sys/fs/cgroup/cpuacct", "rw,nosuid,nodev,noexec,relatime", opt(), "cgroup", "cgroup", "rw,cpuacct"},
			{27, 21, dev(0, 23), "/", "/sys/fs/cgroup/memory", "rw,nosuid,nodev,noexec,relatime", opt(), "cgroup", "cgroup", "rw,memory"},
			{28, 21, dev(0, 24), "/", "/sys/fs/cgroup/devices", "rw,nosuid,nodev,noexec,relatime", opt(), "cgroup", "cgroup", "rw,devices"},
			{29, 21, dev(0, 25), "/", "/sys/fs/cgroup/freezer", "rw,nosuid,nodev,noexec,relatime", opt(), "cgroup", "cgroup", "rw,freezer"},
			{30, 21, dev(0, 26), "/", "/sys/fs/cgroup/net_cls", "rw,nosuid,nodev,noexec,relatime", opt(), "cgroup", "cgroup", "rw,net_cls"},
			{31, 21, dev(0, 27), "/", "/sys/fs/cgroup/blkio", "rw,nosuid,nodev,noexec,relatime", opt(), "cgroup", "cgroup", "rw,blkio"},
			{32, 16, dev(0, 28), "/", "/sys/kernel/security", "rw,relatime", opt(), "autofs", "systemd-1", "rw,fd=22,pgrp=1,timeout=300,minproto=5,maxproto=5,direct"},
			{33, 17, dev(0, 29), "/", "/dev/hugepages", "rw,relatime", opt(), "autofs", "systemd-1", "rw,fd=23,pgrp=1,timeout=300,minproto=5,maxproto=5,direct"},
			{34, 16, dev(0, 30), "/", "/sys/kernel/debug", "rw,relatime", opt(), "autofs", "systemd-1", "rw,fd=24,pgrp=1,timeout=300,minproto=5,maxproto=5,direct"},
			{35, 15, dev(0, 31), "/", "/proc/sys/fs/binfmt_misc", "rw,relatime", opt(), "autofs", "systemd-1", "rw,fd=25,pgrp=1,timeout=300,minproto=5,maxproto=5,direct"},
			{36, 17, dev(0, 32), "/", "/dev/mqueue", "rw,relatime", opt(), "autofs", "systemd-1", "rw,fd=26,pgrp=1,timeout=300,minproto=5,maxproto=5,direct"},
			{37, 15, dev(0, 14), "/", "/proc/bus/usb", "rw,relatime", opt(), "usbfs", "/proc/bus/usb", "rw"},
			{38, 33, dev(0, 33), "/", "/dev/hugepages", "rw,relatime", opt(), "hugetlbfs", "hugetlbfs", "rw"},
			{39, 36, dev(0, 12), "/", "/dev/mqueue", "rw,relatime", opt(), "mqueue", "mqueue", "rw"},
			{40, 20, dev(8, 6), "/", "/boot", "rw,noatime", opt(), "ext3", "/dev/sda6", "rw,errors=continue,barrier=0,data=ordered"},
			{41, 20, dev(253, 0), "/", "/home/kzak", "rw,noatime", opt(), "ext4", "/dev/mapper/kzak-home", "rw,barrier=1,data=ordered"},
			{42, 35, dev(0, 34), "/", "/proc/sys/fs/binfmt_misc", "rw,relatime", opt(), "binfmt_misc", "none", "rw"},
			{43, 16, dev(0, 35), "/", "/sys/fs/fuse/connections", "rw,relatime", opt(), "fusectl", "fusectl", "rw"},
			{44, 41, dev(0, 36), "/", "/home/kzak/.gvfs", "rw,nosuid,nodev,relatime", opt(), "fuse.gvfs-fuse-daemon", "gvfs-fuse-daemon", "rw,user_id=500,group_id=500"},
			{45, 20, dev(0, 37), "/", "/var/lib/nfs/rpc_pipefs", "rw,relatime", opt(), "rpc_pipefs", "sunrpc", "rw"},
			{47, 20, dev(0, 38), "/", "/mnt/sounds", "rw,relatime", opt(), "cifs", "//foo.home/bar/", "rw,unc=\\\\foo.home\\bar,username=kzak,domain=SRGROUP,uid=0,noforceuid,gid=0,noforcegid,addr=192.168.111.1,posixpaths,serverino,acl,rsize=16384,wsize=57344"},
			{49, 20, dev(0, 56), "/", "/mnt/test/foobar", "rw,relatime", opt("shared:323"), "tmpfs", "tmpfs", "rw"},
		}},

		{"sample nosrc", sampleMountinfoNoSrc, nil, "", []vfs.MountInfoEntry{
			{15, 20, dev(0, 3), "/", "/proc", "rw,relatime", opt(), "proc", "/proc", "rw"},
			{16, 20, dev(0, 15), "/", "/sys", "rw,relatime", opt(), "sysfs", "/sys", "rw"},
			{17, 20, dev(0, 5), "/", "/dev", "rw,relatime", opt(), "devtmpfs", "udev", "rw,size=1983516k,nr_inodes=495879,mode=755"},
			{18, 17, dev(0, 10), "/", "/dev/pts", "rw,relatime", opt(), "devpts", "devpts", "rw,gid=5,mode=620,ptmxmode=000"},
			{19, 17, dev(0, 16), "/", "/dev/shm", "rw,relatime", opt(), "tmpfs", "tmpfs", "rw"},
			{20, 1, dev(8, 4), "/", "/", "rw,noatime", opt(), "ext3", "/dev/sda4", "rw,errors=continue,user_xattr,acl,barrier=0,data=ordered"},
			{21, 20, dev(0, 53), "/", "/mnt/test", "rw,relatime", opt("shared:212"), "tmpfs", "", "rw"},
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

				if !reflect.DeepEqual(cur.MountInfoEntry, tc.want[i]) {
					t.Errorf("ParseMountInfo: entry %d\ngot:  %#v\nwant: %#v",
						i, cur.MountInfoEntry, tc.want[i])
				}

				i++
			}

			if i != len(tc.want) {
				t.Errorf("ParseMountInfo: got %d entries, want %d", i, len(tc.want))
			}
		})
	}
}

func dev(maj, min int) vfs.DevT { return vfs.DevT{maj, min} }
func opt(field ...string) []string {
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
20 1 8:4 / / rw,noatime - ext3 /dev/sda4 rw,errors=continue,user_xattr,acl,barrier=0,data=ordered`

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
