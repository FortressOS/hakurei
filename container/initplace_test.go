package container

import (
	"os"
	"testing"

	"hakurei.app/container/check"
	"hakurei.app/container/stub"
)

func TestTmpfileOp(t *testing.T) {
	const sampleDataString = `chronos:x:65534:65534:Hakurei:/var/empty:/bin/zsh`
	var (
		samplePath = check.MustAbs("/etc/passwd")
		sampleData = []byte(sampleDataString)
	)

	checkOpBehaviour(t, []opBehaviourTestCase{
		{"createTemp", &Params{ParentPerm: 0700}, &TmpfileOp{
			Path: samplePath,
			Data: sampleData,
		}, nil, nil, []stub.Call{
			call("createTemp", stub.ExpectArgs{"/", "tmp.*"}, newCheckedFile(t, "tmp.32768", sampleDataString, nil), stub.UniqueError(5)),
		}, stub.UniqueError(5)},

		{"Write", &Params{ParentPerm: 0700}, &TmpfileOp{
			Path: samplePath,
			Data: sampleData,
		}, nil, nil, []stub.Call{
			call("createTemp", stub.ExpectArgs{"/", "tmp.*"}, writeErrOsFile{stub.UniqueError(4)}, nil),
		}, stub.UniqueError(4)},

		{"Close", &Params{ParentPerm: 0700}, &TmpfileOp{
			Path: samplePath,
			Data: sampleData,
		}, nil, nil, []stub.Call{
			call("createTemp", stub.ExpectArgs{"/", "tmp.*"}, newCheckedFile(t, "tmp.32768", sampleDataString, stub.UniqueError(3)), nil),
		}, stub.UniqueError(3)},

		{"ensureFile", &Params{ParentPerm: 0700}, &TmpfileOp{
			Path: samplePath,
			Data: sampleData,
		}, nil, nil, []stub.Call{
			call("createTemp", stub.ExpectArgs{"/", "tmp.*"}, newCheckedFile(t, "tmp.32768", sampleDataString, nil), nil),
			call("ensureFile", stub.ExpectArgs{"/sysroot/etc/passwd", os.FileMode(0444), os.FileMode(0700)}, nil, stub.UniqueError(2)),
		}, stub.UniqueError(2)},

		{"bindMount", &Params{ParentPerm: 0700}, &TmpfileOp{
			Path: samplePath,
			Data: sampleData,
		}, nil, nil, []stub.Call{
			call("createTemp", stub.ExpectArgs{"/", "tmp.*"}, newCheckedFile(t, "tmp.32768", sampleDataString, nil), nil),
			call("ensureFile", stub.ExpectArgs{"/sysroot/etc/passwd", os.FileMode(0444), os.FileMode(0700)}, nil, nil),
			call("bindMount", stub.ExpectArgs{"tmp.32768", "/sysroot/etc/passwd", uintptr(0x5), false}, nil, stub.UniqueError(1)),
		}, stub.UniqueError(1)},

		{"remove", &Params{ParentPerm: 0700}, &TmpfileOp{
			Path: samplePath,
			Data: sampleData,
		}, nil, nil, []stub.Call{
			call("createTemp", stub.ExpectArgs{"/", "tmp.*"}, newCheckedFile(t, "tmp.32768", sampleDataString, nil), nil),
			call("ensureFile", stub.ExpectArgs{"/sysroot/etc/passwd", os.FileMode(0444), os.FileMode(0700)}, nil, nil),
			call("bindMount", stub.ExpectArgs{"tmp.32768", "/sysroot/etc/passwd", uintptr(0x5), false}, nil, nil),
			call("remove", stub.ExpectArgs{"tmp.32768"}, nil, stub.UniqueError(0)),
		}, stub.UniqueError(0)},

		{"success", &Params{ParentPerm: 0700}, &TmpfileOp{
			Path: samplePath,
			Data: sampleData,
		}, nil, nil, []stub.Call{
			call("createTemp", stub.ExpectArgs{"/", "tmp.*"}, newCheckedFile(t, "tmp.32768", sampleDataString, nil), nil),
			call("ensureFile", stub.ExpectArgs{"/sysroot/etc/passwd", os.FileMode(0444), os.FileMode(0700)}, nil, nil),
			call("bindMount", stub.ExpectArgs{"tmp.32768", "/sysroot/etc/passwd", uintptr(0x5), false}, nil, nil),
			call("remove", stub.ExpectArgs{"tmp.32768"}, nil, nil),
		}, nil},
	})

	checkOpsValid(t, []opValidTestCase{
		{"nil", (*TmpfileOp)(nil), false},
		{"zero", new(TmpfileOp), false},
		{"valid", &TmpfileOp{Path: samplePath}, true},
	})

	checkOpsBuilder(t, []opsBuilderTestCase{
		{"noref", new(Ops).Place(samplePath, sampleData), Ops{
			&TmpfileOp{
				Path: samplePath,
				Data: sampleData,
			},
		}},

		{"ref", new(Ops).PlaceP(samplePath, new(*[]byte)), Ops{
			&TmpfileOp{
				Path: samplePath,
				Data: []byte{},
			},
		}},
	})

	checkOpIs(t, []opIsTestCase{
		{"zero", new(TmpfileOp), new(TmpfileOp), false},

		{"differs path", &TmpfileOp{
			Path: check.MustAbs("/etc/group"),
			Data: sampleData,
		}, &TmpfileOp{
			Path: samplePath,
			Data: sampleData,
		}, false},

		{"differs data", &TmpfileOp{
			Path: samplePath,
			Data: append(sampleData, 0),
		}, &TmpfileOp{
			Path: samplePath,
			Data: sampleData,
		}, false},

		{"equals", &TmpfileOp{
			Path: samplePath,
			Data: sampleData,
		}, &TmpfileOp{
			Path: samplePath,
			Data: sampleData,
		}, true},
	})

	checkOpMeta(t, []opMetaTestCase{
		{"passwd", &TmpfileOp{
			Path: samplePath,
			Data: sampleData,
		}, "placing", `tmpfile "/etc/passwd" (49 bytes)`},
	})
}
