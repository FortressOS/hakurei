package container

import (
	"syscall"
	"testing"

	"hakurei.app/container/stub"
)

func TestRemountOp(t *testing.T) {
	checkOpBehaviour(t, []opBehaviourTestCase{
		{"success", new(Params), &RemountOp{
			Target: MustAbs("/"),
			Flags:  syscall.MS_RDONLY,
		}, nil, nil, []stub.Call{
			call("remount", stub.ExpectArgs{"/sysroot", uintptr(1)}, nil, nil),
		}, nil},
	})

	checkOpsValid(t, []opValidTestCase{
		{"nil", (*RemountOp)(nil), false},
		{"zero", new(RemountOp), false},
		{"valid", &RemountOp{Target: MustAbs("/"), Flags: syscall.MS_RDONLY}, true},
	})

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
