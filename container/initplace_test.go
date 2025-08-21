package container

import (
	"os"
	"testing"
)

func TestTmpfileOp(t *testing.T) {
	const sampleDataString = `chronos:x:65534:65534:Hakurei:/var/empty:/bin/zsh`
	var (
		samplePath = MustAbs("/etc/passwd")
		sampleData = []byte(sampleDataString)
	)

	checkOpBehaviour(t, []opBehaviourTestCase{
		{"createTemp", &Params{ParentPerm: 0700}, &TmpfileOp{
			Path: samplePath,
			Data: sampleData,
		}, nil, nil, []kexpect{
			{"createTemp", expectArgs{"/", "tmp.*"}, newCheckedFile(t, "tmp.32768", sampleDataString, nil), errUnique},
		}, wrapErrSelf(errUnique)},

		{"Write", &Params{ParentPerm: 0700}, &TmpfileOp{
			Path: samplePath,
			Data: sampleData,
		}, nil, nil, []kexpect{
			{"createTemp", expectArgs{"/", "tmp.*"}, writeErrOsFile{errUnique}, nil},
		}, wrapErrSuffix(errUnique, "cannot write to intermediate file:")},

		{"Close", &Params{ParentPerm: 0700}, &TmpfileOp{
			Path: samplePath,
			Data: sampleData,
		}, nil, nil, []kexpect{
			{"createTemp", expectArgs{"/", "tmp.*"}, newCheckedFile(t, "tmp.32768", sampleDataString, errUnique), nil},
		}, wrapErrSuffix(errUnique, "cannot close intermediate file:")},

		{"ensureFile", &Params{ParentPerm: 0700}, &TmpfileOp{
			Path: samplePath,
			Data: sampleData,
		}, nil, nil, []kexpect{
			{"createTemp", expectArgs{"/", "tmp.*"}, newCheckedFile(t, "tmp.32768", sampleDataString, nil), nil},
			{"ensureFile", expectArgs{"/sysroot/etc/passwd", os.FileMode(0444), os.FileMode(0700)}, nil, errUnique},
		}, errUnique},

		{"bindMount", &Params{ParentPerm: 0700}, &TmpfileOp{
			Path: samplePath,
			Data: sampleData,
		}, nil, nil, []kexpect{
			{"createTemp", expectArgs{"/", "tmp.*"}, newCheckedFile(t, "tmp.32768", sampleDataString, nil), nil},
			{"ensureFile", expectArgs{"/sysroot/etc/passwd", os.FileMode(0444), os.FileMode(0700)}, nil, nil},
			{"bindMount", expectArgs{"tmp.32768", "/sysroot/etc/passwd", uintptr(0x5), false}, nil, errUnique},
		}, errUnique},

		{"remove", &Params{ParentPerm: 0700}, &TmpfileOp{
			Path: samplePath,
			Data: sampleData,
		}, nil, nil, []kexpect{
			{"createTemp", expectArgs{"/", "tmp.*"}, newCheckedFile(t, "tmp.32768", sampleDataString, nil), nil},
			{"ensureFile", expectArgs{"/sysroot/etc/passwd", os.FileMode(0444), os.FileMode(0700)}, nil, nil},
			{"bindMount", expectArgs{"tmp.32768", "/sysroot/etc/passwd", uintptr(0x5), false}, nil, nil},
			{"remove", expectArgs{"tmp.32768"}, nil, errUnique},
		}, wrapErrSelf(errUnique)},

		{"success", &Params{ParentPerm: 0700}, &TmpfileOp{
			Path: samplePath,
			Data: sampleData,
		}, nil, nil, []kexpect{
			{"createTemp", expectArgs{"/", "tmp.*"}, newCheckedFile(t, "tmp.32768", sampleDataString, nil), nil},
			{"ensureFile", expectArgs{"/sysroot/etc/passwd", os.FileMode(0444), os.FileMode(0700)}, nil, nil},
			{"bindMount", expectArgs{"tmp.32768", "/sysroot/etc/passwd", uintptr(0x5), false}, nil, nil},
			{"remove", expectArgs{"tmp.32768"}, nil, nil},
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
			Path: MustAbs("/etc/group"),
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
