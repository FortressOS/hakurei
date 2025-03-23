package sandbox_test

import (
	"os"
	"path"
	"testing"

	"git.gensokyo.uk/security/fortify/test/sandbox"
)

func TestMounts(t *testing.T) {
	testCases := []struct {
		name string

		sample string
		want   []sandbox.Mntent
	}{
		{"fpkg", `tmpfs / tmpfs rw,nosuid,nodev,relatime,uid=1000002,gid=1000002 0 0
proc /proc proc rw,nosuid,nodev,noexec,relatime 0 0
tmpfs /.fortify tmpfs rw,nosuid,nodev,relatime,size=4k,mode=755,uid=1000002,gid=1000002 0 0
tmpfs /dev tmpfs rw,nosuid,nodev,relatime,mode=755,uid=1000002,gid=1000002 0 0
devtmpfs /dev/null devtmpfs rw,nosuid,size=49396k,nr_inodes=121247,mode=755 0 0
devtmpfs /dev/zero devtmpfs rw,nosuid,size=49396k,nr_inodes=121247,mode=755 0 0
devtmpfs /dev/full devtmpfs rw,nosuid,size=49396k,nr_inodes=121247,mode=755 0 0
devtmpfs /dev/random devtmpfs rw,nosuid,size=49396k,nr_inodes=121247,mode=755 0 0
devtmpfs /dev/urandom devtmpfs rw,nosuid,size=49396k,nr_inodes=121247,mode=755 0 0
devtmpfs /dev/tty devtmpfs rw,nosuid,size=49396k,nr_inodes=121247,mode=755 0 0
devpts /dev/pts devpts rw,nosuid,noexec,relatime,mode=620,ptmxmode=666 0 0
mqueue /dev/mqueue mqueue rw,relatime 0 0
/dev/disk/by-label/nixos /nix/store ext4 ro,nosuid,nodev,relatime 0 0
/dev/disk/by-label/nixos /.fortify/app ext4 ro,nosuid,nodev,relatime 0 0
/dev/disk/by-label/nixos /etc/resolv.conf ext4 ro,nosuid,nodev,relatime 0 0
sysfs /sys/block sysfs ro,nosuid,nodev,noexec,relatime 0 0
sysfs /sys/bus sysfs ro,nosuid,nodev,noexec,relatime 0 0
sysfs /sys/class sysfs ro,nosuid,nodev,noexec,relatime 0 0
sysfs /sys/dev sysfs ro,nosuid,nodev,noexec,relatime 0 0
sysfs /sys/devices sysfs ro,nosuid,nodev,noexec,relatime 0 0
/dev/disk/by-label/nixos /.fortify/nixGL ext4 ro,nosuid,nodev,relatime 0 0
devtmpfs /dev/dri devtmpfs rw,nosuid,size=49396k,nr_inodes=121247,mode=755 0 0
/dev/disk/by-label/nixos /.fortify/etc ext4 ro,nosuid,nodev,relatime 0 0
tmpfs /run/user tmpfs rw,nosuid,nodev,relatime,size=1024k,mode=755,uid=1000002,gid=1000002 0 0
tmpfs /run/user/65534 tmpfs rw,nosuid,nodev,relatime,size=8192k,mode=755,uid=1000002,gid=1000002 0 0
/dev/disk/by-label/nixos /tmp ext4 rw,nosuid,nodev,relatime 0 0
/dev/disk/by-label/nixos /data/data/org.codeberg.dnkl.foot ext4 rw,nosuid,nodev,relatime 0 0
tmpfs /etc/passwd tmpfs ro,nosuid,nodev,relatime,uid=1000002,gid=1000002 0 0
tmpfs /etc/group tmpfs ro,nosuid,nodev,relatime,uid=1000002,gid=1000002 0 0
/dev/disk/by-label/nixos /run/user/65534/wayland-0 ext4 ro,nosuid,nodev,relatime 0 0
tmpfs /run/user/65534/pulse/native tmpfs ro,nosuid,nodev,relatime,size=98784k,nr_inodes=24696,mode=700,uid=1000,gid=100 0 0
/dev/disk/by-label/nixos /run/user/65534/bus ext4 ro,nosuid,nodev,relatime 0 0
overlay /.fortify/sbin/fortify overlay ro,nosuid,nodev,relatime,lowerdir=/mnt-root/nix/.ro-store,upperdir=/mnt-root/nix/.rw-store/upper,workdir=/mnt-root/nix/.rw-store/work,uuid=on 0 0
`, []sandbox.Mntent{
			{"tmpfs", "/", "tmpfs", "rw,nosuid,nodev,relatime,uid=1000002,gid=1000002", 0, 0},
			{"proc", "/proc", "proc", "rw,nosuid,nodev,noexec,relatime", 0, 0},
			{"tmpfs", "/.fortify", "tmpfs", "rw,nosuid,nodev,relatime,size=4k,mode=755,uid=1000002,gid=1000002", 0, 0},
			{"tmpfs", "/dev", "tmpfs", "rw,nosuid,nodev,relatime,mode=755,uid=1000002,gid=1000002", 0, 0},
			{"devtmpfs", "/dev/null", "devtmpfs", "rw,nosuid,size=49396k,nr_inodes=121247,mode=755", 0, 0},
			{"devtmpfs", "/dev/zero", "devtmpfs", "rw,nosuid,size=49396k,nr_inodes=121247,mode=755", 0, 0},
			{"devtmpfs", "/dev/full", "devtmpfs", "rw,nosuid,size=49396k,nr_inodes=121247,mode=755", 0, 0},
			{"devtmpfs", "/dev/random", "devtmpfs", "rw,nosuid,size=49396k,nr_inodes=121247,mode=755", 0, 0},
			{"devtmpfs", "/dev/urandom", "devtmpfs", "rw,nosuid,size=49396k,nr_inodes=121247,mode=755", 0, 0},
			{"devtmpfs", "/dev/tty", "devtmpfs", "rw,nosuid,size=49396k,nr_inodes=121247,mode=755", 0, 0},
			{"devpts", "/dev/pts", "devpts", "rw,nosuid,noexec,relatime,mode=620,ptmxmode=666", 0, 0},
			{"mqueue", "/dev/mqueue", "mqueue", "rw,relatime", 0, 0},
			{"/dev/disk/by-label/nixos", "/nix/store", "ext4", "ro,nosuid,nodev,relatime", 0, 0},
			{"/dev/disk/by-label/nixos", "/.fortify/app", "ext4", "ro,nosuid,nodev,relatime", 0, 0},
			{"/dev/disk/by-label/nixos", "/etc/resolv.conf", "ext4", "ro,nosuid,nodev,relatime", 0, 0},
			{"sysfs", "/sys/block", "sysfs", "ro,nosuid,nodev,noexec,relatime", 0, 0},
			{"sysfs", "/sys/bus", "sysfs", "ro,nosuid,nodev,noexec,relatime", 0, 0},
			{"sysfs", "/sys/class", "sysfs", "ro,nosuid,nodev,noexec,relatime", 0, 0},
			{"sysfs", "/sys/dev", "sysfs", "ro,nosuid,nodev,noexec,relatime", 0, 0},
			{"sysfs", "/sys/devices", "sysfs", "ro,nosuid,nodev,noexec,relatime", 0, 0},
			{"/dev/disk/by-label/nixos", "/.fortify/nixGL", "ext4", "ro,nosuid,nodev,relatime", 0, 0},
			{"devtmpfs", "/dev/dri", "devtmpfs", "rw,nosuid,size=49396k,nr_inodes=121247,mode=755", 0, 0},
			{"/dev/disk/by-label/nixos", "/.fortify/etc", "ext4", "ro,nosuid,nodev,relatime", 0, 0},
			{"tmpfs", "/run/user", "tmpfs", "rw,nosuid,nodev,relatime,size=1024k,mode=755,uid=1000002,gid=1000002", 0, 0},
			{"tmpfs", "/run/user/65534", "tmpfs", "rw,nosuid,nodev,relatime,size=8192k,mode=755,uid=1000002,gid=1000002", 0, 0},
			{"/dev/disk/by-label/nixos", "/tmp", "ext4", "rw,nosuid,nodev,relatime", 0, 0},
			{"/dev/disk/by-label/nixos", "/data/data/org.codeberg.dnkl.foot", "ext4", "rw,nosuid,nodev,relatime", 0, 0},
			{"tmpfs", "/etc/passwd", "tmpfs", "ro,nosuid,nodev,relatime,uid=1000002,gid=1000002", 0, 0},
			{"tmpfs", "/etc/group", "tmpfs", "ro,nosuid,nodev,relatime,uid=1000002,gid=1000002", 0, 0},
			{"/dev/disk/by-label/nixos", "/run/user/65534/wayland-0", "ext4", "ro,nosuid,nodev,relatime", 0, 0},
			{"tmpfs", "/run/user/65534/pulse/native", "tmpfs", "ro,nosuid,nodev,relatime,size=98784k,nr_inodes=24696,mode=700,uid=1000,gid=100", 0, 0},
			{"/dev/disk/by-label/nixos", "/run/user/65534/bus", "ext4", "ro,nosuid,nodev,relatime", 0, 0},
			{"overlay", "/.fortify/sbin/fortify", "overlay", "ro,nosuid,nodev,relatime,lowerdir=/mnt-root/nix/.ro-store,upperdir=/mnt-root/nix/.rw-store/upper,workdir=/mnt-root/nix/.rw-store/work,uuid=on", 0, 0},
		}},
	}

	for _, tc := range testCases {
		name := path.Join(t.TempDir(), "sample")
		if err := os.WriteFile(name, []byte(tc.sample), 0400); err != nil {
			t.Fatalf("cannot write sample: %v", err)
		}

		t.Run(tc.name, func(t *testing.T) {
			f, err := sandbox.OpenMounts(name)
			if err != nil {
				t.Fatalf("OpenMounts: error = %v", err)
			}

			i := 0
			for e := range f.Entries() {
				if i == len(tc.want) {
					t.Errorf("Entries: got more than %d entries", i)
					t.FailNow()
				}
				if *e != tc.want[i] {
					t.Errorf("Entries: entry %d\n got: %s\nwant: %s", i,
						e, &tc.want[i])
					t.FailNow()
				}

				i++
			}

			if err = f.Err(); err != nil {
				t.Fatalf("MountsFile: error = %v", err)
			}
		})

		if err := os.Remove(name); err != nil {
			t.Fatalf("cannot remove %q: %v", name, err)
		}
	}
}
