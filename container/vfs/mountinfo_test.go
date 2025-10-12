package vfs_test

import (
	"encoding/json"
	"errors"
	"iter"
	"os"
	"path"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"hakurei.app/container/vfs"
)

func TestDecoderError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		err     *vfs.DecoderError
		want    string
		target  error
		targetF error
	}{
		{"errno", &vfs.DecoderError{Op: "parse", Line: 0xdeadbeef, Err: syscall.ENOTRECOVERABLE},
			"parse mountinfo at line 3735928559: state not recoverable", syscall.ENOTRECOVERABLE, syscall.EROFS},

		{"strconv", &vfs.DecoderError{Op: "parse", Line: 0xdeadbeef, Err: &strconv.NumError{Func: "Atoi", Num: "meow", Err: strconv.ErrSyntax}},
			`parse mountinfo at line 3735928559: numeric field "meow" invalid syntax`, strconv.ErrSyntax, os.ErrInvalid},

		{"unfold", &vfs.DecoderError{Op: "unfold", Line: -1, Err: vfs.UnfoldTargetError("/proc/nonexistent")},
			"unfold mountinfo: mount point /proc/nonexistent never appeared in mountinfo", vfs.UnfoldTargetError("/proc/nonexistent"), os.ErrNotExist},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			t.Run("error", func(t *testing.T) {
				t.Parallel()
				if got := tc.err.Error(); got != tc.want {
					t.Errorf("Error: %s, want %s", got, tc.want)
				}
			})

			t.Run("is", func(t *testing.T) {
				t.Parallel()
				if !errors.Is(tc.err, tc.target) {
					t.Errorf("Is: unexpected false")
				}
				if errors.Is(tc.err, tc.targetF) {
					t.Errorf("Is: unexpected true")
				}
			})
		})
	}
}

func TestMountInfo(t *testing.T) {
	t.Parallel()

	testCases := []mountInfoTest{
		{"count", sampleMountinfoBase + `
21 20 0:53/ /mnt/test rw,relatime - tmpfs  rw
21 16 0:17 / /sys/fs/cgroup rw,nosuid,nodev,noexec,relatime - tmpfs tmpfs rw,mode=755`,
			&vfs.DecoderError{Op: "parse", Line: 6, Err: vfs.ErrMountInfoFields},
			"", nil, nil, nil},

		{"sep", sampleMountinfoBase + `
21 20 0:53 / /mnt/test rw,relatime shared:212 _ tmpfs  rw
21 16 0:17 / /sys/fs/cgroup rw,nosuid,nodev,noexec,relatime - tmpfs tmpfs rw,mode=755`,
			&vfs.DecoderError{Op: "parse", Line: 6, Err: vfs.ErrMountInfoSep},
			"", nil, nil, nil},

		{"id", sampleMountinfoBase + `
id 20 0:53 / /mnt/test rw,relatime shared:212 - tmpfs  rw
21 16 0:17 / /sys/fs/cgroup rw,nosuid,nodev,noexec,relatime - tmpfs tmpfs rw,mode=755`,
			&vfs.DecoderError{Op: "parse", Line: 6, Err: &strconv.NumError{Func: "Atoi", Num: "id", Err: strconv.ErrSyntax}},
			"", nil, nil, nil},

		{"parent", sampleMountinfoBase + `
21 parent 0:53 / /mnt/test rw,relatime shared:212 - tmpfs  rw
21 16 0:17 / /sys/fs/cgroup rw,nosuid,nodev,noexec,relatime - tmpfs tmpfs rw,mode=755`,
			&vfs.DecoderError{Op: "parse", Line: 6, Err: &strconv.NumError{Func: "Atoi", Num: "parent", Err: strconv.ErrSyntax}}, "", nil, nil, nil},

		{"devno", sampleMountinfoBase + `
21 20 053 / /mnt/test rw,relatime shared:212 - tmpfs  rw
21 16 0:17 / /sys/fs/cgroup rw,nosuid,nodev,noexec,relatime - tmpfs tmpfs rw,mode=755`,
			nil, "parse mountinfo at line 6: unexpected EOF", nil, nil, nil},

		{"maj", sampleMountinfoBase + `
21 20 maj:53 / /mnt/test rw,relatime shared:212 - tmpfs  rw
21 16 0:17 / /sys/fs/cgroup rw,nosuid,nodev,noexec,relatime - tmpfs tmpfs rw,mode=755`,
			nil, "parse mountinfo at line 6: expected integer", nil, nil, nil},

		{"min", sampleMountinfoBase + `
21 20 0:min / /mnt/test rw,relatime shared:212 - tmpfs  rw
21 16 0:17 / /sys/fs/cgroup rw,nosuid,nodev,noexec,relatime - tmpfs tmpfs rw,mode=755`,
			nil, "parse mountinfo at line 6: expected integer", nil, nil, nil},

		{"mountroot", sampleMountinfoBase + `
21 20 0:53  /mnt/test rw,relatime - tmpfs  rw
21 16 0:17 / /sys/fs/cgroup rw,nosuid,nodev,noexec,relatime - tmpfs tmpfs rw,mode=755`,
			&vfs.DecoderError{Op: "parse", Line: 6, Err: vfs.ErrMountInfoEmpty}, "", nil, nil, nil},

		{"target", sampleMountinfoBase + `
21 20 0:53 /  rw,relatime - tmpfs  rw
21 16 0:17 / /sys/fs/cgroup rw,nosuid,nodev,noexec,relatime - tmpfs tmpfs rw,mode=755`,
			&vfs.DecoderError{Op: "parse", Line: 6, Err: vfs.ErrMountInfoEmpty}, "", nil, nil, nil},

		{"vfs options", sampleMountinfoBase + `
21 20 0:53 / /mnt/test  - tmpfs  rw
21 16 0:17 / /sys/fs/cgroup rw,nosuid,nodev,noexec,relatime - tmpfs tmpfs rw,mode=755`,
			&vfs.DecoderError{Op: "parse", Line: 6, Err: vfs.ErrMountInfoEmpty}, "", nil, nil, nil},

		{"FS type", sampleMountinfoBase + `
21 16 0:17 / /sys/fs/cgroup rw,nosuid,nodev,noexec,relatime - tmpfs tmpfs rw,mode=755
21 20 0:53 / /mnt/test rw,relatime -   rw`,
			&vfs.DecoderError{Op: "parse", Line: 7, Err: vfs.ErrMountInfoEmpty}, "", nil, nil, nil},

		{"base", sampleMountinfoBase, nil, "", []*wantMountInfo{
			m(15, 20, 0, 3, "/", "/proc", "rw,relatime", o(), "proc", "/proc", "rw", syscall.MS_RELATIME, nil),
			m(16, 20, 0, 15, "/", "/sys", "rw,relatime", o(), "sysfs", "/sys", "rw", syscall.MS_RELATIME, nil),
			m(17, 20, 0, 5, "/", "/dev", "rw,relatime", o(), "devtmpfs", "udev", "rw,size=1983516k,nr_inodes=495879,mode=755", syscall.MS_RELATIME, nil),
			m(18, 17, 0, 10, "/", "/dev/pts", "rw,relatime", o(), "devpts", "devpts", "rw,gid=5,mode=620,ptmxmode=000", syscall.MS_RELATIME, nil),
			m(19, 17, 0, 16, "/", "/dev/shm", "rw,relatime", o(), "tmpfs", "tmpfs", "rw", syscall.MS_RELATIME, nil),
			m(20, 1, 8, 4, "/", "/", "ro,noatime,nodiratime,meow", o(), "ext3", "/dev/sda4", "rw,errors=continue,user_xattr,acl,barrier=0,data=ordered", syscall.MS_RDONLY|syscall.MS_NOATIME|syscall.MS_NODIRATIME, []string{"meow"}),
		},
			mn(20, 1, 8, 4, "/", "/", "ro,noatime,nodiratime,meow", o(), "ext3", "/dev/sda4", "rw,errors=continue,user_xattr,acl,barrier=0,data=ordered", false,
				mn(15, 20, 0, 3, "/", "/proc", "rw,relatime", o(), "proc", "/proc", "rw", false, nil,
					mn(16, 20, 0, 15, "/", "/sys", "rw,relatime", o(), "sysfs", "/sys", "rw", false, nil,
						mn(17, 20, 0, 5, "/", "/dev", "rw,relatime", o(), "devtmpfs", "udev", "rw,size=1983516k,nr_inodes=495879,mode=755", false,
							mn(18, 17, 0, 10, "/", "/dev/pts", "rw,relatime", o(), "devpts", "devpts", "rw,gid=5,mode=620,ptmxmode=000", false, nil,
								mn(19, 17, 0, 16, "/", "/dev/shm", "rw,relatime", o(), "tmpfs", "tmpfs", "rw", false, nil, nil)),
							nil))), nil), func(n *vfs.MountInfoNode) []*vfs.MountInfoNode {
				return []*vfs.MountInfoNode{
					n,
					n.FirstChild,
					n.FirstChild.NextSibling,
					n.FirstChild.NextSibling.NextSibling,
					n.FirstChild.NextSibling.NextSibling.FirstChild,
					n.FirstChild.NextSibling.NextSibling.FirstChild.NextSibling,
				}
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
			m(49, 20, 0, 56, "/", "/mnt/test/foobar", "rw,relatime,nosymfollow", o("shared:323"), "tmpfs", "tmpfs", "rw", syscall.MS_RELATIME|vfs.MS_NOSYMFOLLOW, nil),
		}, nil, nil},

		{"sample nosrc", sampleMountinfoNoSrc, nil, "", []*wantMountInfo{
			m(15, 20, 0, 3, "/", "/proc", "rw,relatime", o(), "proc", "/proc", "rw", syscall.MS_RELATIME, nil),
			m(16, 20, 0, 15, "/", "/sys", "rw,relatime", o(), "sysfs", "/sys", "rw", syscall.MS_RELATIME, nil),
			m(17, 20, 0, 5, "/", "/dev", "rw,relatime", o(), "devtmpfs", "udev", "rw,size=1983516k,nr_inodes=495879,mode=755", syscall.MS_RELATIME, nil),
			m(18, 17, 0, 10, "/", "/dev/pts", "rw,relatime", o(), "devpts", "devpts", "rw,gid=5,mode=620,ptmxmode=000", syscall.MS_RELATIME, nil),
			m(19, 17, 0, 16, "/", "/dev/shm", "rw,relatime", o(), "tmpfs", "tmpfs", "rw", syscall.MS_RELATIME, nil),
			m(20, 1, 8, 4, "/", "/", "rw,noatime", o(), "ext3", "/dev/sda4", "rw,errors=continue,user_xattr,acl,barrier=0,data=ordered", syscall.MS_NOATIME, nil),
			m(21, 20, 0, 53, "/", "/mnt/test", "rw,relatime", o("shared:212"), "tmpfs", "", "rw", syscall.MS_RELATIME, nil),
		}, nil, nil},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			t.Run("decode", func(t *testing.T) {
				t.Parallel()
				var got *vfs.MountInfo
				d := vfs.NewMountInfoDecoder(strings.NewReader(tc.sample))
				err := d.Decode(&got)
				tc.check(t, d, "Decode",
					func(yield func(*vfs.MountInfoEntry) bool) {
						for cur := got; cur != nil; cur = cur.Next {
							if !yield(&cur.MountInfoEntry) {
								return
							}
						}
					}, func() error { return err })
				t.Run("reuse", func(t *testing.T) {
					tc.check(t, d, "Entries",
						d.Entries(), d.Err)
				})
			})

			t.Run("iter", func(t *testing.T) {
				t.Parallel()
				d := vfs.NewMountInfoDecoder(strings.NewReader(tc.sample))
				tc.check(t, d, "Entries",
					d.Entries(), d.Err)

				t.Run("reuse", func(t *testing.T) {
					tc.check(t, d, "Entries",
						d.Entries(), d.Err)
				})
			})

			t.Run("yield", func(t *testing.T) {
				t.Parallel()
				d := vfs.NewMountInfoDecoder(strings.NewReader(tc.sample))
				v := false
				d.Entries()(func(entry *vfs.MountInfoEntry) bool { v = !v; return v })
				d.Entries()(func(entry *vfs.MountInfoEntry) bool { return false })

				tc.check(t, d, "Entries",
					d.Entries(), d.Err)

				t.Run("reuse", func(t *testing.T) {
					tc.check(t, d, "Entries",
						d.Entries(), d.Err)
				})
			})
		})
	}
}

type mountInfoTest struct {
	name      string
	sample    string
	wantErr   error
	wantError string
	want      []*wantMountInfo

	wantNode     *vfs.MountInfoNode
	wantCollectF func(n *vfs.MountInfoNode) []*vfs.MountInfoNode
}

func (tc *mountInfoTest) check(t *testing.T, d *vfs.MountInfoDecoder, funcName string,
	got iter.Seq[*vfs.MountInfoEntry], gotErr func() error) {
	i := 0
	for cur := range got {
		if i == len(tc.want) {
			if funcName != "Decode" && (tc.wantErr != nil || tc.wantError != "") {
				continue
			}

			t.Errorf("%s: got more than %d entries", funcName, len(tc.want))
			break
		}

		if !reflect.DeepEqual(cur, &tc.want[i].MountInfoEntry) {
			t.Errorf("%s: entry %d\ngot:  %#v\nwant: %#v",
				funcName, i, cur, tc.want[i])
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
		t.Errorf("%s: got %d entries, want %d", funcName, i, len(tc.want))
	}

	if tc.wantErr == nil && tc.wantError == "" && tc.wantCollectF != nil {
		t.Run("unfold", func(t *testing.T) {
			n, err := d.Unfold("/")
			if err != nil {
				t.Errorf("Unfold: error = %v", err)
			} else {
				t.Run("stop", func(t *testing.T) {
					v := false
					n.Collective()(func(node *vfs.MountInfoNode) bool { v = !v; return v })
				})

				if !reflect.DeepEqual(n, tc.wantNode) {
					t.Errorf("Unfold: %s, want %s",
						mustMarshal(n), mustMarshal(tc.wantNode))
				}

				t.Run("collective", func(t *testing.T) {
					wantCollect := tc.wantCollectF(n)
					if gotCollect := slices.Collect(n.Collective()); !reflect.DeepEqual(gotCollect, wantCollect) {
						t.Errorf("Collective: \ngot  %#v\nwant %#v",
							gotCollect, wantCollect)
					}
				})
			}
		})
	} else if tc.wantNode != nil || tc.wantCollectF != nil {
		panic("invalid test case")
	} else if _, err := d.Unfold("/"); !reflect.DeepEqual(err, tc.wantErr) {
		if tc.wantError == "" {
			t.Errorf("Unfold: error = %#v, wantErr %#v",
				err, tc.wantErr)
		} else if err != nil && err.Error() != tc.wantError {
			t.Errorf("Unfold: error = %q, wantError %q",
				err, tc.wantError)
		}
	}

	if err := gotErr(); !reflect.DeepEqual(err, tc.wantErr) {
		if tc.wantError == "" {
			t.Errorf("%s: error = %#v, wantErr %#v",
				funcName, err, tc.wantErr)
		} else if err != nil && err.Error() != tc.wantError {
			t.Errorf("%s: error = %q, wantError %q",
				funcName, err, tc.wantError)
		}
	}
}

func mustMarshal(v any) string {
	p, err := json.Marshal(v)
	if err != nil {
		panic(err.Error())
	}
	return string(p)
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

func mn(
	id, parent, maj, min int, root, target, vfsOptstr string, optFields []string, fsType, source, fsOptstr string,
	covered bool, firstChild, nextSibling *vfs.MountInfoNode,
) *vfs.MountInfoNode {
	return &vfs.MountInfoNode{
		MountInfoEntry: &vfs.MountInfoEntry{
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
		},
		FirstChild:  firstChild,
		NextSibling: nextSibling,
		Clean:       path.Clean(target),
		Covered:     covered,
	}
}

func o(field ...string) []string {
	if field == nil {
		return []string{}
	}
	return field
}

const (
	sampleMountinfoBase = `15 20 0:3 / /proc rw,relatime - proc /proc rw
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
49 20 0:56 / /mnt/test/foobar rw,relatime,nosymfollow shared:323 - tmpfs tmpfs rw`

	sampleMountinfoNoSrc = `15 20 0:3 / /proc rw,relatime - proc /proc rw
16 20 0:15 / /sys rw,relatime - sysfs /sys rw
17 20 0:5 / /dev rw,relatime - devtmpfs udev rw,size=1983516k,nr_inodes=495879,mode=755
18 17 0:10 / /dev/pts rw,relatime - devpts devpts rw,gid=5,mode=620,ptmxmode=000
19 17 0:16 / /dev/shm rw,relatime - tmpfs tmpfs rw
20 1 8:4 / / rw,noatime - ext3 /dev/sda4 rw,errors=continue,user_xattr,acl,barrier=0,data=ordered
21 20 0:53 / /mnt/test rw,relatime shared:212 - tmpfs  rw`
)
