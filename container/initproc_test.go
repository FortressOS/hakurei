package container

import (
	"os"
	"testing"

	"hakurei.app/container/check"
	"hakurei.app/container/stub"
)

func TestMountProcOp(t *testing.T) {
	t.Parallel()

	checkOpBehaviour(t, []opBehaviourTestCase{
		{"mkdir", &Params{ParentPerm: 0755},
			&MountProcOp{
				Target: check.MustAbs("/proc/"),
			}, nil, nil, []stub.Call{
				call("mkdirAll", stub.ExpectArgs{"/sysroot/proc", os.FileMode(0755)}, nil, stub.UniqueError(0)),
			}, stub.UniqueError(0)},

		{"success", &Params{ParentPerm: 0700},
			&MountProcOp{
				Target: check.MustAbs("/proc/"),
			}, nil, nil, []stub.Call{
				call("mkdirAll", stub.ExpectArgs{"/sysroot/proc", os.FileMode(0700)}, nil, nil),
				call("mount", stub.ExpectArgs{"proc", "/sysroot/proc", "proc", uintptr(0xe), ""}, nil, nil),
			}, nil},
	})

	checkOpsValid(t, []opValidTestCase{
		{"nil", (*MountProcOp)(nil), false},
		{"zero", new(MountProcOp), false},
		{"valid", &MountProcOp{Target: check.MustAbs("/proc/")}, true},
	})

	checkOpsBuilder(t, []opsBuilderTestCase{
		{"proc", new(Ops).Proc(check.MustAbs("/proc/")), Ops{
			&MountProcOp{Target: check.MustAbs("/proc/")},
		}},
	})

	checkOpIs(t, []opIsTestCase{
		{"zero", new(MountProcOp), new(MountProcOp), false},

		{"target differs", &MountProcOp{
			Target: check.MustAbs("/proc/nonexistent"),
		}, &MountProcOp{
			Target: check.MustAbs("/proc/"),
		}, false},

		{"equals", &MountProcOp{
			Target: check.MustAbs("/proc/"),
		}, &MountProcOp{
			Target: check.MustAbs("/proc/"),
		}, true},
	})

	checkOpMeta(t, []opMetaTestCase{
		{"proc", &MountProcOp{Target: check.MustAbs("/proc/")},
			"mounting", `proc on "/proc/"`},
	})
}
