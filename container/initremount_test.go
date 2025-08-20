package container

import (
	"syscall"
	"testing"
)

func TestRemountOp(t *testing.T) {
	checkOpsBuilder(t, []opsBuilderTestCase{
		{"root", new(Ops).Remount(MustAbs("/"), syscall.MS_RDONLY), Ops{
			&RemountOp{
				Target: MustAbs("/"),
				Flags:  syscall.MS_RDONLY,
			},
		}},
	})

	checkOpIs(t, []opIsTestCase{
		{"zero", new(RemountOp), new(RemountOp), false},

		{"target differs", &RemountOp{
			Target: MustAbs("/dev/"),
			Flags:  syscall.MS_RDONLY,
		}, &RemountOp{
			Target: MustAbs("/"),
			Flags:  syscall.MS_RDONLY,
		}, false},

		{"flags differs", &RemountOp{
			Target: MustAbs("/"),
			Flags:  syscall.MS_RDONLY | syscall.MS_NODEV,
		}, &RemountOp{
			Target: MustAbs("/"),
			Flags:  syscall.MS_RDONLY,
		}, false},

		{"equals", &RemountOp{
			Target: MustAbs("/"),
			Flags:  syscall.MS_RDONLY,
		}, &RemountOp{
			Target: MustAbs("/"),
			Flags:  syscall.MS_RDONLY,
		}, true},
	})

	checkOpMeta(t, []opMetaTestCase{
		{"root", &RemountOp{
			Target: MustAbs("/"),
			Flags:  syscall.MS_RDONLY,
		}, "remounting", `"/" flags 0x1`},
	})
}
