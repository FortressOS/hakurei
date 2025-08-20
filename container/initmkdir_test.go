package container

import "testing"

func TestMkdirOp(t *testing.T) {
	checkOpsBuilder(t, []opsBuilderTestCase{
		{"etc", new(Ops).Mkdir(MustAbs("/etc/"), 0), Ops{
			&MkdirOp{Path: MustAbs("/etc/")},
		}},
	})

	checkOpIs(t, []opIsTestCase{
		{"zero", new(MkdirOp), new(MkdirOp), false},
		{"path differs", &MkdirOp{Path: MustAbs("/"), Perm: 0755}, &MkdirOp{Path: MustAbs("/etc/"), Perm: 0755}, false},
		{"perm differs", &MkdirOp{Path: MustAbs("/")}, &MkdirOp{Path: MustAbs("/"), Perm: 0755}, false},
		{"equals", &MkdirOp{Path: MustAbs("/")}, &MkdirOp{Path: MustAbs("/")}, true},
	})

	checkOpMeta(t, []opMetaTestCase{
		{"etc", &MkdirOp{
			Path: MustAbs("/etc/"),
		}, "creating", `directory "/etc/" perm ----------`},
	})
}
