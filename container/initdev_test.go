package container

import "testing"

func TestMountDevOp(t *testing.T) {
	checkOpsBuilder(t, []opsBuilderTestCase{
		{"dev", new(Ops).Dev(MustAbs("/dev/"), true), Ops{
			&MountDevOp{
				Target: MustAbs("/dev/"),
				Mqueue: true,
			},
		}},

		{"dev writable", new(Ops).DevWritable(MustAbs("/.hakurei/dev/"), false), Ops{
			&MountDevOp{
				Target: MustAbs("/.hakurei/dev/"),
				Write:  true,
			},
		}},
	})

	checkOpIs(t, []opIsTestCase{
		{"zero", new(MountDevOp), new(MountDevOp), false},

		{"write differs", &MountDevOp{
			Target: MustAbs("/dev/"),
			Mqueue: true,
		}, &MountDevOp{
			Target: MustAbs("/dev/"),
			Mqueue: true,
			Write:  true,
		}, false},

		{"mqueue differs", &MountDevOp{
			Target: MustAbs("/dev/"),
		}, &MountDevOp{
			Target: MustAbs("/dev/"),
			Mqueue: true,
		}, false},

		{"target differs", &MountDevOp{
			Target: MustAbs("/"),
			Mqueue: true,
		}, &MountDevOp{
			Target: MustAbs("/dev/"),
			Mqueue: true,
		}, false},

		{"equals", &MountDevOp{
			Target: MustAbs("/dev/"),
			Mqueue: true,
		}, &MountDevOp{
			Target: MustAbs("/dev/"),
			Mqueue: true,
		}, true},
	})

	checkOpMeta(t, []opMetaTestCase{
		{"mqueue", &MountDevOp{
			Target: MustAbs("/dev/"),
			Mqueue: true,
		}, "mounting", `dev on "/dev/" with mqueue`},

		{"dev", &MountDevOp{
			Target: MustAbs("/dev/"),
		}, "mounting", `dev on "/dev/"`},
	})
}
