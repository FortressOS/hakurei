package container

import "testing"

func TestAutoEtcOp(t *testing.T) {
	checkOpsValid(t, []opValidTestCase{
		{"nil", (*AutoEtcOp)(nil), false},
		{"zero", new(AutoEtcOp), true},
		{"populated", &AutoEtcOp{Prefix: ":3"}, true},
	})

	checkOpsBuilder(t, []opsBuilderTestCase{
		{"pd", new(Ops).Etc(MustAbs("/etc/"), "048090b6ed8f9ebb10e275ff5d8c0659"), Ops{
			&MkdirOp{Path: MustAbs("/etc/"), Perm: 0755},
			&BindMountOp{
				Source: MustAbs("/etc/"),
				Target: MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
			},
			&AutoEtcOp{Prefix: "048090b6ed8f9ebb10e275ff5d8c0659"},
		}},
	})

	checkOpIs(t, []opIsTestCase{
		{"zero", new(AutoEtcOp), new(AutoEtcOp), true},
		{"differs", &AutoEtcOp{Prefix: "\x00"}, &AutoEtcOp{":3"}, false},
		{"equals", &AutoEtcOp{Prefix: ":3"}, &AutoEtcOp{":3"}, true},
	})

	checkOpMeta(t, []opMetaTestCase{
		{"etc", &AutoEtcOp{
			Prefix: ":3",
		}, "setting up", "auto etc :3"},
	})

	t.Run("host path rel", func(t *testing.T) {
		op := &AutoEtcOp{Prefix: "048090b6ed8f9ebb10e275ff5d8c0659"}
		wantHostPath := "/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"
		wantHostRel := ".host/048090b6ed8f9ebb10e275ff5d8c0659"
		if got := op.hostPath(); got.String() != wantHostPath {
			t.Errorf("hostPath: %q, want %q", got, wantHostPath)
		}
		if got := op.hostRel(); got != wantHostRel {
			t.Errorf("hostRel: %q, want %q", got, wantHostRel)
		}
	})
}
