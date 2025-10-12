package container

import (
	"os"
	"testing"

	"hakurei.app/container/check"
	"hakurei.app/container/stub"
)

func TestMkdirOp(t *testing.T) {
	t.Parallel()

	checkOpBehaviour(t, []opBehaviourTestCase{
		{"success", new(Params), &MkdirOp{
			Path: check.MustAbs("/.hakurei"),
			Perm: 0500,
		}, nil, nil, []stub.Call{
			call("mkdirAll", stub.ExpectArgs{"/sysroot/.hakurei", os.FileMode(0500)}, nil, nil),
		}, nil},
	})

	checkOpsValid(t, []opValidTestCase{
		{"nil", (*MkdirOp)(nil), false},
		{"zero", new(MkdirOp), false},
		{"valid", &MkdirOp{Path: check.MustAbs("/.hakurei")}, true},
	})

	checkOpsBuilder(t, []opsBuilderTestCase{
		{"etc", new(Ops).Mkdir(check.MustAbs("/etc/"), 0), Ops{
			&MkdirOp{Path: check.MustAbs("/etc/")},
		}},
	})

	checkOpIs(t, []opIsTestCase{
		{"zero", new(MkdirOp), new(MkdirOp), false},
		{"path differs", &MkdirOp{Path: check.MustAbs("/"), Perm: 0755}, &MkdirOp{Path: check.MustAbs("/etc/"), Perm: 0755}, false},
		{"perm differs", &MkdirOp{Path: check.MustAbs("/")}, &MkdirOp{Path: check.MustAbs("/"), Perm: 0755}, false},
		{"equals", &MkdirOp{Path: check.MustAbs("/")}, &MkdirOp{Path: check.MustAbs("/")}, true},
	})

	checkOpMeta(t, []opMetaTestCase{
		{"etc", &MkdirOp{
			Path: check.MustAbs("/etc/"),
		}, "creating", `directory "/etc/" perm ----------`},
	})
}
