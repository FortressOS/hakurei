package container

import (
	"os"
	"testing"

	"hakurei.app/container/stub"
)

func TestMkdirOp(t *testing.T) {
	checkOpBehaviour(t, []opBehaviourTestCase{
		{"success", new(Params), &MkdirOp{
			Path: MustAbs("/.hakurei"),
			Perm: 0500,
		}, nil, nil, []stub.Call{
			call("mkdirAll", stub.ExpectArgs{"/sysroot/.hakurei", os.FileMode(0500)}, nil, nil),
		}, nil},
	})

	checkOpsValid(t, []opValidTestCase{
		{"nil", (*MkdirOp)(nil), false},
		{"zero", new(MkdirOp), false},
		{"valid", &MkdirOp{Path: MustAbs("/.hakurei")}, true},
	})

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
