package container

import (
	"os"
	"testing"

	"hakurei.app/container/stub"
)

func TestMountProcOp(t *testing.T) {
	checkOpBehaviour(t, []opBehaviourTestCase{
		{"mkdir", &Params{ParentPerm: 0755},
			&MountProcOp{
				Target: MustAbs("/proc/"),
			}, nil, nil, []stub.Call{
				{"mkdirAll", stub.ExpectArgs{"/sysroot/proc", os.FileMode(0755)}, nil, stub.UniqueError(0)},
			}, stub.UniqueError(0)},

		{"success", &Params{ParentPerm: 0700},
			&MountProcOp{
				Target: MustAbs("/proc/"),
			}, nil, nil, []stub.Call{
				{"mkdirAll", stub.ExpectArgs{"/sysroot/proc", os.FileMode(0700)}, nil, nil},
				{"mount", stub.ExpectArgs{"proc", "/sysroot/proc", "proc", uintptr(0xe), ""}, nil, nil},
			}, nil},
	})

	checkOpsValid(t, []opValidTestCase{
		{"nil", (*MountProcOp)(nil), false},
		{"zero", new(MountProcOp), false},
		{"valid", &MountProcOp{Target: MustAbs("/proc/")}, true},
	})

	checkOpsBuilder(t, []opsBuilderTestCase{
		{"proc", new(Ops).Proc(MustAbs("/proc/")), Ops{
			&MountProcOp{Target: MustAbs("/proc/")},
		}},
	})

	checkOpIs(t, []opIsTestCase{
		{"zero", new(MountProcOp), new(MountProcOp), false},

		{"target differs", &MountProcOp{
			Target: MustAbs("/proc/nonexistent"),
		}, &MountProcOp{
			Target: MustAbs("/proc/"),
		}, false},

		{"equals", &MountProcOp{
			Target: MustAbs("/proc/"),
		}, &MountProcOp{
			Target: MustAbs("/proc/"),
		}, true},
	})

	checkOpMeta(t, []opMetaTestCase{
		{"proc", &MountProcOp{Target: MustAbs("/proc/")},
			"mounting", `proc on "/proc/"`},
	})
}
