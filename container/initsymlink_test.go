package container

import (
	"os"
	"testing"
)

func TestSymlinkOp(t *testing.T) {
	checkOpBehaviour(t, []opBehaviourTestCase{
		{"mkdir", &Params{ParentPerm: 0700}, &SymlinkOp{
			Target:   MustAbs("/etc/nixos"),
			LinkName: "/etc/static/nixos",
		}, nil, nil, []kexpect{
			{"mkdirAll", expectArgs{"/sysroot/etc", os.FileMode(0700)}, nil, errUnique},
		}, wrapErrSelf(errUnique)},

		{"abs", &Params{ParentPerm: 0755}, &SymlinkOp{
			Target:      MustAbs("/etc/mtab"),
			LinkName:    "etc/mtab",
			Dereference: true,
		}, nil, &AbsoluteError{"etc/mtab"}, nil, nil},

		{"readlink", &Params{ParentPerm: 0755}, &SymlinkOp{
			Target:      MustAbs("/etc/mtab"),
			LinkName:    "/etc/mtab",
			Dereference: true,
		}, []kexpect{
			{"readlink", expectArgs{"/etc/mtab"}, "/proc/mounts", errUnique},
		}, wrapErrSelf(errUnique), nil, nil},

		{"success noderef", &Params{ParentPerm: 0700}, &SymlinkOp{
			Target:   MustAbs("/etc/nixos"),
			LinkName: "/etc/static/nixos",
		}, nil, nil, []kexpect{
			{"mkdirAll", expectArgs{"/sysroot/etc", os.FileMode(0700)}, nil, nil},
			{"symlink", expectArgs{"/etc/static/nixos", "/sysroot/etc/nixos"}, nil, nil},
		}, nil},

		{"success", &Params{ParentPerm: 0755}, &SymlinkOp{
			Target:      MustAbs("/etc/mtab"),
			LinkName:    "/etc/mtab",
			Dereference: true,
		}, []kexpect{
			{"readlink", expectArgs{"/etc/mtab"}, "/proc/mounts", nil},
		}, nil, []kexpect{
			{"mkdirAll", expectArgs{"/sysroot/etc", os.FileMode(0755)}, nil, nil},
			{"symlink", expectArgs{"/proc/mounts", "/sysroot/etc/mtab"}, nil, nil},
		}, nil},
	})

	checkOpsValid(t, []opValidTestCase{
		{"nil", (*SymlinkOp)(nil), false},
		{"zero", new(SymlinkOp), false},
		{"nil target", &SymlinkOp{LinkName: "/run/current-system"}, false},
		{"zero linkname", &SymlinkOp{Target: MustAbs("/run/current-system")}, false},
		{"valid", &SymlinkOp{Target: MustAbs("/run/current-system"), LinkName: "/run/current-system", Dereference: true}, true},
	})

	checkOpsBuilder(t, []opsBuilderTestCase{
		{"current-system", new(Ops).Link(
			MustAbs("/run/current-system"),
			"/run/current-system",
			true,
		), Ops{
			&SymlinkOp{
				Target:      MustAbs("/run/current-system"),
				LinkName:    "/run/current-system",
				Dereference: true,
			},
		}},
	})

	checkOpIs(t, []opIsTestCase{
		{"zero", new(SymlinkOp), new(SymlinkOp), false},

		{"target differs", &SymlinkOp{
			Target:      MustAbs("/run/current-system/differs"),
			LinkName:    "/run/current-system",
			Dereference: true,
		}, &SymlinkOp{
			Target:      MustAbs("/run/current-system"),
			LinkName:    "/run/current-system",
			Dereference: true,
		}, false},

		{"linkname differs", &SymlinkOp{
			Target:      MustAbs("/run/current-system"),
			LinkName:    "/run/current-system/differs",
			Dereference: true,
		}, &SymlinkOp{
			Target:      MustAbs("/run/current-system"),
			LinkName:    "/run/current-system",
			Dereference: true,
		}, false},

		{"dereference differs", &SymlinkOp{
			Target:   MustAbs("/run/current-system"),
			LinkName: "/run/current-system",
		}, &SymlinkOp{
			Target:      MustAbs("/run/current-system"),
			LinkName:    "/run/current-system",
			Dereference: true,
		}, false},

		{"equals", &SymlinkOp{
			Target:      MustAbs("/run/current-system"),
			LinkName:    "/run/current-system",
			Dereference: true,
		}, &SymlinkOp{
			Target:      MustAbs("/run/current-system"),
			LinkName:    "/run/current-system",
			Dereference: true,
		}, true},
	})

	checkOpMeta(t, []opMetaTestCase{
		{"current-system", &SymlinkOp{
			Target:      MustAbs("/run/current-system"),
			LinkName:    "/run/current-system",
			Dereference: true,
		}, "creating", `symlink on "/run/current-system" linkname "/run/current-system"`},
	})
}
