package container

import "testing"

func TestMountOverlayOp(t *testing.T) {
	checkOpsValid(t, []opValidTestCase{
		{"nil", (*MountOverlayOp)(nil), false},
		{"zero", new(MountOverlayOp), false},
		{"nil lower", &MountOverlayOp{Target: MustAbs("/"), Lower: []*Absolute{nil}}, false},
		{"ro", &MountOverlayOp{Target: MustAbs("/"), Lower: []*Absolute{MustAbs("/")}}, true},
		{"ro work", &MountOverlayOp{Target: MustAbs("/"), Work: MustAbs("/tmp/")}, false},
		{"rw", &MountOverlayOp{Target: MustAbs("/"), Lower: []*Absolute{MustAbs("/")}, Upper: MustAbs("/"), Work: MustAbs("/")}, true},
	})

	checkOpsBuilder(t, []opsBuilderTestCase{
		{"full", new(Ops).Overlay(
			MustAbs("/nix/store"),
			MustAbs("/mnt-root/nix/.rw-store/upper"),
			MustAbs("/mnt-root/nix/.rw-store/work"),
			MustAbs("/mnt-root/nix/.ro-store"),
		), Ops{
			&MountOverlayOp{
				Target: MustAbs("/nix/store"),
				Lower:  []*Absolute{MustAbs("/mnt-root/nix/.ro-store")},
				Upper:  MustAbs("/mnt-root/nix/.rw-store/upper"),
				Work:   MustAbs("/mnt-root/nix/.rw-store/work"),
			},
		}},

		{"ephemeral", new(Ops).OverlayEphemeral(MustAbs("/nix/store"), MustAbs("/mnt-root/nix/.ro-store")), Ops{
			&MountOverlayOp{
				Target: MustAbs("/nix/store"),
				Lower:  []*Absolute{MustAbs("/mnt-root/nix/.ro-store")},
				Upper:  MustAbs("/"),
			},
		}},

		{"readonly", new(Ops).OverlayReadonly(MustAbs("/nix/store"), MustAbs("/mnt-root/nix/.ro-store")), Ops{
			&MountOverlayOp{
				Target: MustAbs("/nix/store"),
				Lower:  []*Absolute{MustAbs("/mnt-root/nix/.ro-store")},
			},
		}},
	})

	checkOpIs(t, []opIsTestCase{
		{"zero", new(MountOverlayOp), new(MountOverlayOp), false},

		{"differs target", &MountOverlayOp{
			Target: MustAbs("/nix/store/differs"),
			Lower:  []*Absolute{MustAbs("/mnt-root/nix/.ro-store")},
			Upper:  MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:   MustAbs("/mnt-root/nix/.rw-store/work"),
		}, &MountOverlayOp{
			Target: MustAbs("/nix/store"),
			Lower:  []*Absolute{MustAbs("/mnt-root/nix/.ro-store")},
			Upper:  MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:   MustAbs("/mnt-root/nix/.rw-store/work")}, false},

		{"differs lower", &MountOverlayOp{
			Target: MustAbs("/nix/store"),
			Lower:  []*Absolute{MustAbs("/mnt-root/nix/.ro-store/differs")},
			Upper:  MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:   MustAbs("/mnt-root/nix/.rw-store/work"),
		}, &MountOverlayOp{
			Target: MustAbs("/nix/store"),
			Lower:  []*Absolute{MustAbs("/mnt-root/nix/.ro-store")},
			Upper:  MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:   MustAbs("/mnt-root/nix/.rw-store/work")}, false},

		{"differs upper", &MountOverlayOp{
			Target: MustAbs("/nix/store"),
			Lower:  []*Absolute{MustAbs("/mnt-root/nix/.ro-store")},
			Upper:  MustAbs("/mnt-root/nix/.rw-store/upper/differs"),
			Work:   MustAbs("/mnt-root/nix/.rw-store/work"),
		}, &MountOverlayOp{
			Target: MustAbs("/nix/store"),
			Lower:  []*Absolute{MustAbs("/mnt-root/nix/.ro-store")},
			Upper:  MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:   MustAbs("/mnt-root/nix/.rw-store/work")}, false},

		{"differs work", &MountOverlayOp{
			Target: MustAbs("/nix/store"),
			Lower:  []*Absolute{MustAbs("/mnt-root/nix/.ro-store")},
			Upper:  MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:   MustAbs("/mnt-root/nix/.rw-store/work/differs"),
		}, &MountOverlayOp{
			Target: MustAbs("/nix/store"),
			Lower:  []*Absolute{MustAbs("/mnt-root/nix/.ro-store")},
			Upper:  MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:   MustAbs("/mnt-root/nix/.rw-store/work")}, false},

		{"equals ro", &MountOverlayOp{
			Target: MustAbs("/nix/store"),
			Lower:  []*Absolute{MustAbs("/mnt-root/nix/.ro-store")},
		}, &MountOverlayOp{
			Target: MustAbs("/nix/store"),
			Lower:  []*Absolute{MustAbs("/mnt-root/nix/.ro-store")}}, true},

		{"equals", &MountOverlayOp{
			Target: MustAbs("/nix/store"),
			Lower:  []*Absolute{MustAbs("/mnt-root/nix/.ro-store")},
			Upper:  MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:   MustAbs("/mnt-root/nix/.rw-store/work"),
		}, &MountOverlayOp{
			Target: MustAbs("/nix/store"),
			Lower:  []*Absolute{MustAbs("/mnt-root/nix/.ro-store")},
			Upper:  MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:   MustAbs("/mnt-root/nix/.rw-store/work")}, true},
	})

	checkOpMeta(t, []opMetaTestCase{
		{"nix", &MountOverlayOp{
			Target: MustAbs("/nix/store"),
			Lower:  []*Absolute{MustAbs("/mnt-root/nix/.ro-store")},
			Upper:  MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:   MustAbs("/mnt-root/nix/.rw-store/work"),
		}, "mounting", `overlay on "/nix/store" with 1 layers`},
	})
}
