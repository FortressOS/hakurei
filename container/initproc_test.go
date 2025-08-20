package container

import "testing"

func TestMountProcOp(t *testing.T) {
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
