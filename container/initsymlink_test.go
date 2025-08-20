package container

import "testing"

func TestSymlinkOp(t *testing.T) {
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
