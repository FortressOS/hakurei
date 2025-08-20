package container

import (
	"syscall"
	"testing"
)

func TestMountTmpfsOp(t *testing.T) {
	checkOpsValid(t, []opValidTestCase{
		{"nil", (*MountTmpfsOp)(nil), false},
		{"zero", new(MountTmpfsOp), false},
		{"nil path", &MountTmpfsOp{FSName: "tmpfs"}, false},
		{"zero fsname", &MountTmpfsOp{Path: MustAbs("/tmp/")}, false},
		{"valid", &MountTmpfsOp{FSName: "tmpfs", Path: MustAbs("/tmp/")}, true},
	})

	checkOpsBuilder(t, []opsBuilderTestCase{
		{"runtime", new(Ops).Tmpfs(
			MustAbs("/run/user"),
			1<<10,
			0755,
		), Ops{
			&MountTmpfsOp{
				FSName: "ephemeral",
				Path:   MustAbs("/run/user"),
				Flags:  syscall.MS_NOSUID | syscall.MS_NODEV,
				Size:   1 << 10,
				Perm:   0755,
			},
		}},

		{"nscd", new(Ops).Readonly(
			MustAbs("/var/run/nscd"),
			0755,
		), Ops{
			&MountTmpfsOp{
				FSName: "readonly",
				Path:   MustAbs("/var/run/nscd"),
				Flags:  syscall.MS_NOSUID | syscall.MS_NODEV | syscall.MS_RDONLY,
				Perm:   0755,
			},
		}},
	})

	checkOpIs(t, []opIsTestCase{
		{"zero", new(MountTmpfsOp), new(MountTmpfsOp), false},

		{"fsname differs", &MountTmpfsOp{
			FSName: "readonly",
			Path:   MustAbs("/run/user"),
			Flags:  syscall.MS_NOSUID | syscall.MS_NODEV,
			Size:   1 << 10,
			Perm:   0755,
		}, &MountTmpfsOp{
			FSName: "ephemeral",
			Path:   MustAbs("/run/user"),
			Flags:  syscall.MS_NOSUID | syscall.MS_NODEV,
			Size:   1 << 10,
			Perm:   0755,
		}, false},

		{"path differs", &MountTmpfsOp{
			FSName: "ephemeral",
			Path:   MustAbs("/run/user/differs"),
			Flags:  syscall.MS_NOSUID | syscall.MS_NODEV,
			Size:   1 << 10,
			Perm:   0755,
		}, &MountTmpfsOp{
			FSName: "ephemeral",
			Path:   MustAbs("/run/user"),
			Flags:  syscall.MS_NOSUID | syscall.MS_NODEV,
			Size:   1 << 10,
			Perm:   0755,
		}, false},

		{"flags differs", &MountTmpfsOp{
			FSName: "ephemeral",
			Path:   MustAbs("/run/user"),
			Flags:  syscall.MS_NOSUID | syscall.MS_NODEV | syscall.MS_RDONLY,
			Size:   1 << 10,
			Perm:   0755,
		}, &MountTmpfsOp{
			FSName: "ephemeral",
			Path:   MustAbs("/run/user"),
			Flags:  syscall.MS_NOSUID | syscall.MS_NODEV,
			Size:   1 << 10,
			Perm:   0755,
		}, false},

		{"size differs", &MountTmpfsOp{
			FSName: "ephemeral",
			Path:   MustAbs("/run/user"),
			Flags:  syscall.MS_NOSUID | syscall.MS_NODEV,
			Size:   1,
			Perm:   0755,
		}, &MountTmpfsOp{
			FSName: "ephemeral",
			Path:   MustAbs("/run/user"),
			Flags:  syscall.MS_NOSUID | syscall.MS_NODEV,
			Size:   1 << 10,
			Perm:   0755,
		}, false},

		{"perm differs", &MountTmpfsOp{
			FSName: "ephemeral",
			Path:   MustAbs("/run/user"),
			Flags:  syscall.MS_NOSUID | syscall.MS_NODEV,
			Size:   1 << 10,
			Perm:   0700,
		}, &MountTmpfsOp{
			FSName: "ephemeral",
			Path:   MustAbs("/run/user"),
			Flags:  syscall.MS_NOSUID | syscall.MS_NODEV,
			Size:   1 << 10,
			Perm:   0755,
		}, false},

		{"equals", &MountTmpfsOp{
			FSName: "ephemeral",
			Path:   MustAbs("/run/user"),
			Flags:  syscall.MS_NOSUID | syscall.MS_NODEV,
			Size:   1 << 10,
			Perm:   0755,
		}, &MountTmpfsOp{
			FSName: "ephemeral",
			Path:   MustAbs("/run/user"),
			Flags:  syscall.MS_NOSUID | syscall.MS_NODEV,
			Size:   1 << 10,
			Perm:   0755,
		}, true},
	})

	checkOpMeta(t, []opMetaTestCase{
		{"runtime", &MountTmpfsOp{
			FSName: "ephemeral",
			Path:   MustAbs("/run/user"),
			Flags:  syscall.MS_NOSUID | syscall.MS_NODEV,
			Size:   1 << 10,
			Perm:   0755,
		}, "mounting", `tmpfs on "/run/user" size 1024`},
	})
}
