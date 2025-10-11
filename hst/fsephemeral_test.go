package hst_test

import (
	"syscall"
	"testing"

	"hakurei.app/container"
	"hakurei.app/hst"
)

func TestFSEphemeral(t *testing.T) {
	checkFs(t, []fsTestCase{
		{"nil", (*hst.FSEphemeral)(nil), false, nil, nil, nil, "<invalid>"},

		{"full", &hst.FSEphemeral{
			Target: m("/run/user/65534"),
			Write:  true,
			Size:   1 << 10,
			Perm:   0700,
		}, true, container.Ops{&container.MountTmpfsOp{
			FSName: "ephemeral",
			Path:   m("/run/user/65534"),
			Flags:  syscall.MS_NOSUID | syscall.MS_NODEV,
			Size:   1 << 10,
			Perm:   0700,
		}}, m("/run/user/65534"), nil,
			"w+ephemeral(-rwx------):/run/user/65534"},

		{"cover ro", &hst.FSEphemeral{Target: m("/run/nscd")}, true,
			container.Ops{&container.MountTmpfsOp{
				FSName: "readonly",
				Path:   m("/run/nscd"),
				Flags:  syscall.MS_NOSUID | syscall.MS_NODEV | syscall.MS_RDONLY,
				Perm:   0755,
			}}, m("/run/nscd"), nil,
			"+ephemeral(-rwxr-xr-x):/run/nscd"},

		{"negative size", &hst.FSEphemeral{
			Target: hst.AbsPrivateTmp,
			Write:  true,
			Size:   -1,
		}, true, container.Ops{&container.MountTmpfsOp{
			FSName: "ephemeral",
			Path:   hst.AbsPrivateTmp,
			Flags:  syscall.MS_NOSUID | syscall.MS_NODEV,
			Perm:   0755,
		}}, hst.AbsPrivateTmp, nil,
			"w+ephemeral(-rwxr-xr-x):/.hakurei"},
	})
}
