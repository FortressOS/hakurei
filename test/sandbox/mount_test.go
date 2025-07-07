//go:build testtool

package sandbox_test

import (
	"os"
	"path"
	"testing"

	"hakurei.app/test/sandbox"
)

func TestMountinfo(t *testing.T) {
	testCases := []struct {
		name string

		sample string
		want   []*sandbox.MountinfoEntry
	}{
		{"util-linux", `15 20 0:3 / /proc rw,relatime - proc /proc rw
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
49 20 0:56 / /mnt/test/foobar rw,relatime,nosymfollow shared:323 - tmpfs tmpfs rw`, []*sandbox.MountinfoEntry{
			e(15, 20, "/", "/proc", "rw,relatime", "proc", "/proc", "rw"),
			e(16, 20, "/", "/sys", "rw,relatime", "sysfs", "/sys", "rw"),
			e(17, 20, "/", "/dev", "rw,relatime", "devtmpfs", "udev", "rw,size=1983516k,nr_inodes=495879,mode=755"),
			e(18, 17, "/", "/dev/pts", "rw,relatime", "devpts", "devpts", "rw,gid=5,mode=620,ptmxmode=000"),
			e(19, 17, "/", "/dev/shm", "rw,relatime", "tmpfs", "tmpfs", "rw"),
			e(20, 1, "/", "/", "rw,noatime", "ext3", "/dev/sda4", "rw,errors=continue,user_xattr,acl,barrier=0,data=ordered"),
			e(21, 16, "/", "/sys/fs/cgroup", "rw,nosuid,nodev,noexec,relatime", "tmpfs", "tmpfs", "rw,mode=755"),
			e(22, 21, "/", "/sys/fs/cgroup/systemd", "rw,nosuid,nodev,noexec,relatime", "cgroup", "cgroup", "rw,release_agent=/lib/systemd/systemd-cgroups-agent,name=systemd"),
			e(23, 21, "/", "/sys/fs/cgroup/cpuset", "rw,nosuid,nodev,noexec,relatime", "cgroup", "cgroup", "rw,cpuset"),
			e(24, 21, "/", "/sys/fs/cgroup/ns", "rw,nosuid,nodev,noexec,relatime", "cgroup", "cgroup", "rw,ns"),
			e(25, 21, "/", "/sys/fs/cgroup/cpu", "rw,nosuid,nodev,noexec,relatime", "cgroup", "cgroup", "rw,cpu"),
			e(26, 21, "/", "/sys/fs/cgroup/cpuacct", "rw,nosuid,nodev,noexec,relatime", "cgroup", "cgroup", "rw,cpuacct"),
			e(27, 21, "/", "/sys/fs/cgroup/memory", "rw,nosuid,nodev,noexec,relatime", "cgroup", "cgroup", "rw,memory"),
			e(28, 21, "/", "/sys/fs/cgroup/devices", "rw,nosuid,nodev,noexec,relatime", "cgroup", "cgroup", "rw,devices"),
			e(29, 21, "/", "/sys/fs/cgroup/freezer", "rw,nosuid,nodev,noexec,relatime", "cgroup", "cgroup", "rw,freezer"),
			e(30, 21, "/", "/sys/fs/cgroup/net_cls", "rw,nosuid,nodev,noexec,relatime", "cgroup", "cgroup", "rw,net_cls"),
			e(31, 21, "/", "/sys/fs/cgroup/blkio", "rw,nosuid,nodev,noexec,relatime", "cgroup", "cgroup", "rw,blkio"),
			e(32, 16, "/", "/sys/kernel/security", "rw,relatime", "autofs", "systemd-1", "rw,fd=22,pgrp=1,timeout=300,minproto=5,maxproto=5,direct"),
			e(33, 17, "/", "/dev/hugepages", "rw,relatime", "autofs", "systemd-1", "rw,fd=23,pgrp=1,timeout=300,minproto=5,maxproto=5,direct"),
			e(34, 16, "/", "/sys/kernel/debug", "rw,relatime", "autofs", "systemd-1", "rw,fd=24,pgrp=1,timeout=300,minproto=5,maxproto=5,direct"),
			e(35, 15, "/", "/proc/sys/fs/binfmt_misc", "rw,relatime", "autofs", "systemd-1", "rw,fd=25,pgrp=1,timeout=300,minproto=5,maxproto=5,direct"),
			e(36, 17, "/", "/dev/mqueue", "rw,relatime", "autofs", "systemd-1", "rw,fd=26,pgrp=1,timeout=300,minproto=5,maxproto=5,direct"),
			e(37, 15, "/", "/proc/bus/usb", "rw,relatime", "usbfs", "/proc/bus/usb", "rw"),
			e(38, 33, "/", "/dev/hugepages", "rw,relatime", "hugetlbfs", "hugetlbfs", "rw"),
			e(39, 36, "/", "/dev/mqueue", "rw,relatime", "mqueue", "mqueue", "rw"),
			e(40, 20, "/", "/boot", "rw,noatime", "ext3", "/dev/sda6", "rw,errors=continue,barrier=0,data=ordered"),
			e(41, 20, "/", "/home/kzak", "rw,noatime", "ext4", "/dev/mapper/kzak-home", "rw,barrier=1,data=ordered"),
			e(42, 35, "/", "/proc/sys/fs/binfmt_misc", "rw,relatime", "binfmt_misc", "none", "rw"),
			e(43, 16, "/", "/sys/fs/fuse/connections", "rw,relatime", "fusectl", "fusectl", "rw"),
			e(44, 41, "/", "/home/kzak/.gvfs", "rw,nosuid,nodev,relatime", "fuse.gvfs-fuse-daemon", "gvfs-fuse-daemon", "rw,user_id=500,group_id=500"),
			e(45, 20, "/", "/var/lib/nfs/rpc_pipefs", "rw,relatime", "rpc_pipefs", "sunrpc", "rw"),
			e(47, 20, "/", "/mnt/sounds", "rw,relatime", "cifs", "//foo.home/bar/", "rw,unc=\\\\foo.home\\bar,username=kzak,domain=SRGROUP,uid=0,noforceuid,gid=0,noforcegid,addr=192.168.111.1,posixpaths,serverino,acl,rsize=16384,wsize=57344"),
			e(49, 20, "/", "/mnt/test/foobar", "rw,relatime,nosymfollow", "tmpfs", "tmpfs", "rw"),
		}},
	}

	for _, tc := range testCases {
		name := path.Join(t.TempDir(), "sample")
		if err := os.WriteFile(name, []byte(tc.sample), 0400); err != nil {
			t.Fatalf("cannot write sample: %v", err)
		}

		t.Run(tc.name, func(t *testing.T) {
			m := sandbox.NewMountinfo(name)
			if err := m.Parse(); err != nil {
				t.Fatalf("Parse: error = %v", err)
			}

			i := 0
			for ent := range m.Entries() {
				if i == len(tc.want) {
					t.Errorf("Entries: got more than %d entries", i)
					t.FailNow()
				}
				if !ent.EqualWithIgnore(tc.want[i], "\x00") {
					t.Errorf("Entries: entry %d\n got: %#v\nwant: %#v", i,
						ent, &tc.want[i])
					t.FailNow()
				} else {
					t.Logf("%s", ent)
				}

				i++
			}

			if err := m.Err(); err != nil {
				t.Fatalf("Mountinfo: error = %v", err)
			}

			m.Unref()
		})

		if err := os.Remove(name); err != nil {
			t.Fatalf("cannot remove %q: %v", name, err)
		}
	}
}

func e(
	id, parent int, root, target, vfsOptstr string, fsType, source, fsOptstr string,
) *sandbox.MountinfoEntry {
	return &sandbox.MountinfoEntry{
		ID:        id,
		Parent:    parent,
		Root:      root,
		Target:    target,
		VfsOptstr: vfsOptstr,
		FsType:    fsType,
		Source:    source,
		FsOptstr:  fsOptstr,
	}
}
