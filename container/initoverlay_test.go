package container

import (
	"errors"
	"io/fs"
	"os"
	"testing"
)

func TestMountOverlayOp(t *testing.T) {
	checkOpBehaviour(t, []opBehaviourTestCase{
		{"mkdirTemp invalid ephemeral", &Params{ParentPerm: 0705}, &MountOverlayOp{
			Target: MustAbs("/"),
			Lower: []*Absolute{
				MustAbs("/var/lib/planterette/base/debian:f92c9052"),
				MustAbs("/var/lib/planterette/app/org.chromium.Chromium@debian:f92c9052"),
			},
			Upper: MustAbs("/proc/"),
		}, nil, msg.WrapErr(fs.ErrInvalid, `upperdir has unexpected value "/proc/"`), nil, nil},

		{"mkdirTemp upper ephemeral", &Params{ParentPerm: 0705}, &MountOverlayOp{
			Target: MustAbs("/"),
			Lower: []*Absolute{
				MustAbs("/var/lib/planterette/base/debian:f92c9052"),
				MustAbs("/var/lib/planterette/app/org.chromium.Chromium@debian:f92c9052"),
			},
			Upper: MustAbs("/"),
		}, []kexpect{
			{"evalSymlinks", expectArgs{"/var/lib/planterette/base/debian:f92c9052"}, "/var/lib/planterette/base/debian:f92c9052", nil},
			{"evalSymlinks", expectArgs{"/var/lib/planterette/app/org.chromium.Chromium@debian:f92c9052"}, "/var/lib/planterette/app/org.chromium.Chromium@debian:f92c9052", nil},
		}, nil, []kexpect{
			{"mkdirAll", expectArgs{"/sysroot", os.FileMode(0705)}, nil, nil},
			{"mkdirTemp", expectArgs{"/", "overlay.upper.*"}, "overlay.upper.32768", errUnique},
		}, wrapErrSelf(errUnique)},

		{"mkdirTemp work ephemeral", &Params{ParentPerm: 0705}, &MountOverlayOp{
			Target: MustAbs("/"),
			Lower: []*Absolute{
				MustAbs("/var/lib/planterette/base/debian:f92c9052"),
				MustAbs("/var/lib/planterette/app/org.chromium.Chromium@debian:f92c9052"),
			},
			Upper: MustAbs("/"),
		}, []kexpect{
			{"evalSymlinks", expectArgs{"/var/lib/planterette/base/debian:f92c9052"}, "/var/lib/planterette/base/debian:f92c9052", nil},
			{"evalSymlinks", expectArgs{"/var/lib/planterette/app/org.chromium.Chromium@debian:f92c9052"}, "/var/lib/planterette/app/org.chromium.Chromium@debian:f92c9052", nil},
		}, nil, []kexpect{
			{"mkdirAll", expectArgs{"/sysroot", os.FileMode(0705)}, nil, nil},
			{"mkdirTemp", expectArgs{"/", "overlay.upper.*"}, "overlay.upper.32768", nil},
			{"mkdirTemp", expectArgs{"/", "overlay.work.*"}, "overlay.work.32768", errUnique},
		}, wrapErrSelf(errUnique)},

		{"success ephemeral", &Params{ParentPerm: 0705}, &MountOverlayOp{
			Target: MustAbs("/"),
			Lower: []*Absolute{
				MustAbs("/var/lib/planterette/base/debian:f92c9052"),
				MustAbs("/var/lib/planterette/app/org.chromium.Chromium@debian:f92c9052"),
			},
			Upper: MustAbs("/"),
		}, []kexpect{
			{"evalSymlinks", expectArgs{"/var/lib/planterette/base/debian:f92c9052"}, "/var/lib/planterette/base/debian:f92c9052", nil},
			{"evalSymlinks", expectArgs{"/var/lib/planterette/app/org.chromium.Chromium@debian:f92c9052"}, "/var/lib/planterette/app/org.chromium.Chromium@debian:f92c9052", nil},
		}, nil, []kexpect{
			{"mkdirAll", expectArgs{"/sysroot", os.FileMode(0705)}, nil, nil},
			{"mkdirTemp", expectArgs{"/", "overlay.upper.*"}, "overlay.upper.32768", nil},
			{"mkdirTemp", expectArgs{"/", "overlay.work.*"}, "overlay.work.32768", nil},
			{"mount", expectArgs{"overlay", "/sysroot", "overlay", uintptr(0), "" +
				"upperdir=overlay.upper.32768," +
				"workdir=overlay.work.32768," +
				"lowerdir=" +
				`/host/var/lib/planterette/base/debian\:f92c9052:` +
				`/host/var/lib/planterette/app/org.chromium.Chromium@debian\:f92c9052,` +
				"userxattr"}, nil, nil},
		}, nil},

		{"short lower ro", &Params{ParentPerm: 0755}, &MountOverlayOp{
			Target: MustAbs("/nix/store"),
			Lower: []*Absolute{
				MustAbs("/mnt-root/nix/.ro-store"),
			},
		}, []kexpect{
			{"evalSymlinks", expectArgs{"/mnt-root/nix/.ro-store"}, "/mnt-root/nix/.ro-store", nil},
		}, nil, []kexpect{
			{"mkdirAll", expectArgs{"/sysroot/nix/store", os.FileMode(0755)}, nil, nil},
		}, msg.WrapErr(fs.ErrInvalid, "readonly overlay requires at least two lowerdir")},

		{"success ro", &Params{ParentPerm: 0755}, &MountOverlayOp{
			Target: MustAbs("/nix/store"),
			Lower: []*Absolute{
				MustAbs("/mnt-root/nix/.ro-store"),
				MustAbs("/mnt-root/nix/.ro-store0"),
			},
		}, []kexpect{
			{"evalSymlinks", expectArgs{"/mnt-root/nix/.ro-store"}, "/mnt-root/nix/.ro-store", nil},
			{"evalSymlinks", expectArgs{"/mnt-root/nix/.ro-store0"}, "/mnt-root/nix/.ro-store0", nil},
		}, nil, []kexpect{
			{"mkdirAll", expectArgs{"/sysroot/nix/store", os.FileMode(0755)}, nil, nil},
			{"mount", expectArgs{"overlay", "/sysroot/nix/store", "overlay", uintptr(0), "" +
				"lowerdir=" +
				"/host/mnt-root/nix/.ro-store:" +
				"/host/mnt-root/nix/.ro-store0," +
				"userxattr"}, nil, nil},
		}, nil},

		{"nil lower", &Params{ParentPerm: 0700}, &MountOverlayOp{
			Target: MustAbs("/nix/store"),
			Upper:  MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:   MustAbs("/mnt-root/nix/.rw-store/work"),
		}, []kexpect{
			{"evalSymlinks", expectArgs{"/mnt-root/nix/.rw-store/upper"}, "/mnt-root/nix/.rw-store/.upper", nil},
			{"evalSymlinks", expectArgs{"/mnt-root/nix/.rw-store/work"}, "/mnt-root/nix/.rw-store/.work", nil},
		}, nil, []kexpect{
			{"mkdirAll", expectArgs{"/sysroot/nix/store", os.FileMode(0700)}, nil, nil},
		}, msg.WrapErr(fs.ErrInvalid, "overlay requires at least one lowerdir")},

		{"evalSymlinks upper", &Params{ParentPerm: 0700}, &MountOverlayOp{
			Target: MustAbs("/nix/store"),
			Lower:  []*Absolute{MustAbs("/mnt-root/nix/.ro-store")},
			Upper:  MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:   MustAbs("/mnt-root/nix/.rw-store/work"),
		}, []kexpect{
			{"evalSymlinks", expectArgs{"/mnt-root/nix/.rw-store/upper"}, "/mnt-root/nix/.rw-store/.upper", errUnique},
		}, wrapErrSelf(errUnique), nil, nil},

		{"evalSymlinks work", &Params{ParentPerm: 0700}, &MountOverlayOp{
			Target: MustAbs("/nix/store"),
			Lower:  []*Absolute{MustAbs("/mnt-root/nix/.ro-store")},
			Upper:  MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:   MustAbs("/mnt-root/nix/.rw-store/work"),
		}, []kexpect{
			{"evalSymlinks", expectArgs{"/mnt-root/nix/.rw-store/upper"}, "/mnt-root/nix/.rw-store/.upper", nil},
			{"evalSymlinks", expectArgs{"/mnt-root/nix/.rw-store/work"}, "/mnt-root/nix/.rw-store/.work", errUnique},
		}, wrapErrSelf(errUnique), nil, nil},

		{"evalSymlinks lower", &Params{ParentPerm: 0700}, &MountOverlayOp{
			Target: MustAbs("/nix/store"),
			Lower:  []*Absolute{MustAbs("/mnt-root/nix/.ro-store")},
			Upper:  MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:   MustAbs("/mnt-root/nix/.rw-store/work"),
		}, []kexpect{
			{"evalSymlinks", expectArgs{"/mnt-root/nix/.rw-store/upper"}, "/mnt-root/nix/.rw-store/.upper", nil},
			{"evalSymlinks", expectArgs{"/mnt-root/nix/.rw-store/work"}, "/mnt-root/nix/.rw-store/.work", nil},
			{"evalSymlinks", expectArgs{"/mnt-root/nix/.ro-store"}, "/mnt-root/nix/ro-store", errUnique},
		}, wrapErrSelf(errUnique), nil, nil},

		{"mkdirAll", &Params{ParentPerm: 0700}, &MountOverlayOp{
			Target: MustAbs("/nix/store"),
			Lower:  []*Absolute{MustAbs("/mnt-root/nix/.ro-store")},
			Upper:  MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:   MustAbs("/mnt-root/nix/.rw-store/work"),
		}, []kexpect{
			{"evalSymlinks", expectArgs{"/mnt-root/nix/.rw-store/upper"}, "/mnt-root/nix/.rw-store/.upper", nil},
			{"evalSymlinks", expectArgs{"/mnt-root/nix/.rw-store/work"}, "/mnt-root/nix/.rw-store/.work", nil},
			{"evalSymlinks", expectArgs{"/mnt-root/nix/.ro-store"}, "/mnt-root/nix/ro-store", nil},
		}, nil, []kexpect{
			{"mkdirAll", expectArgs{"/sysroot/nix/store", os.FileMode(0700)}, nil, errUnique},
		}, wrapErrSelf(errUnique)},

		{"mount", &Params{ParentPerm: 0700}, &MountOverlayOp{
			Target: MustAbs("/nix/store"),
			Lower:  []*Absolute{MustAbs("/mnt-root/nix/.ro-store")},
			Upper:  MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:   MustAbs("/mnt-root/nix/.rw-store/work"),
		}, []kexpect{
			{"evalSymlinks", expectArgs{"/mnt-root/nix/.rw-store/upper"}, "/mnt-root/nix/.rw-store/.upper", nil},
			{"evalSymlinks", expectArgs{"/mnt-root/nix/.rw-store/work"}, "/mnt-root/nix/.rw-store/.work", nil},
			{"evalSymlinks", expectArgs{"/mnt-root/nix/.ro-store"}, "/mnt-root/nix/ro-store", nil},
		}, nil, []kexpect{
			{"mkdirAll", expectArgs{"/sysroot/nix/store", os.FileMode(0700)}, nil, nil},
			{"mount", expectArgs{"overlay", "/sysroot/nix/store", "overlay", uintptr(0), "upperdir=/host/mnt-root/nix/.rw-store/.upper,workdir=/host/mnt-root/nix/.rw-store/.work,lowerdir=/host/mnt-root/nix/ro-store,userxattr"}, nil, errUnique},
		}, wrapErrSuffix(errUnique, `cannot mount overlay on "/nix/store":`)},

		{"success single layer", &Params{ParentPerm: 0700}, &MountOverlayOp{
			Target: MustAbs("/nix/store"),
			Lower:  []*Absolute{MustAbs("/mnt-root/nix/.ro-store")},
			Upper:  MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:   MustAbs("/mnt-root/nix/.rw-store/work"),
		}, []kexpect{
			{"evalSymlinks", expectArgs{"/mnt-root/nix/.rw-store/upper"}, "/mnt-root/nix/.rw-store/.upper", nil},
			{"evalSymlinks", expectArgs{"/mnt-root/nix/.rw-store/work"}, "/mnt-root/nix/.rw-store/.work", nil},
			{"evalSymlinks", expectArgs{"/mnt-root/nix/.ro-store"}, "/mnt-root/nix/ro-store", nil},
		}, nil, []kexpect{
			{"mkdirAll", expectArgs{"/sysroot/nix/store", os.FileMode(0700)}, nil, nil},
			{"mount", expectArgs{"overlay", "/sysroot/nix/store", "overlay", uintptr(0), "" +
				"upperdir=/host/mnt-root/nix/.rw-store/.upper," +
				"workdir=/host/mnt-root/nix/.rw-store/.work," +
				"lowerdir=/host/mnt-root/nix/ro-store," +
				"userxattr"}, nil, nil},
		}, nil},

		{"success", &Params{ParentPerm: 0700}, &MountOverlayOp{
			Target: MustAbs("/nix/store"),
			Lower: []*Absolute{
				MustAbs("/mnt-root/nix/.ro-store"),
				MustAbs("/mnt-root/nix/.ro-store0"),
				MustAbs("/mnt-root/nix/.ro-store1"),
				MustAbs("/mnt-root/nix/.ro-store2"),
				MustAbs("/mnt-root/nix/.ro-store3"),
			},
			Upper: MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:  MustAbs("/mnt-root/nix/.rw-store/work"),
		}, []kexpect{
			{"evalSymlinks", expectArgs{"/mnt-root/nix/.rw-store/upper"}, "/mnt-root/nix/.rw-store/.upper", nil},
			{"evalSymlinks", expectArgs{"/mnt-root/nix/.rw-store/work"}, "/mnt-root/nix/.rw-store/.work", nil},
			{"evalSymlinks", expectArgs{"/mnt-root/nix/.ro-store"}, "/mnt-root/nix/ro-store", nil},
			{"evalSymlinks", expectArgs{"/mnt-root/nix/.ro-store0"}, "/mnt-root/nix/ro-store0", nil},
			{"evalSymlinks", expectArgs{"/mnt-root/nix/.ro-store1"}, "/mnt-root/nix/ro-store1", nil},
			{"evalSymlinks", expectArgs{"/mnt-root/nix/.ro-store2"}, "/mnt-root/nix/ro-store2", nil},
			{"evalSymlinks", expectArgs{"/mnt-root/nix/.ro-store3"}, "/mnt-root/nix/ro-store3", nil},
		}, nil, []kexpect{
			{"mkdirAll", expectArgs{"/sysroot/nix/store", os.FileMode(0700)}, nil, nil},
			{"mount", expectArgs{"overlay", "/sysroot/nix/store", "overlay", uintptr(0), "" +
				"upperdir=/host/mnt-root/nix/.rw-store/.upper," +
				"workdir=/host/mnt-root/nix/.rw-store/.work," +
				"lowerdir=" +
				"/host/mnt-root/nix/ro-store:" +
				"/host/mnt-root/nix/ro-store0:" +
				"/host/mnt-root/nix/ro-store1:" +
				"/host/mnt-root/nix/ro-store2:" +
				"/host/mnt-root/nix/ro-store3," +
				"userxattr"}, nil, nil},
		}, nil},
	})

	t.Run("unreachable", func(t *testing.T) {
		t.Run("nil Upper non-nil Work not ephemeral", func(t *testing.T) {
			wantErr := msg.WrapErr(fs.ErrClosed, "impossible overlay state reached")
			if err := (&MountOverlayOp{
				Work: MustAbs("/"),
			}).early(nil, nil); !errors.Is(err, wantErr) {
				t.Errorf("apply: error = %v, want %v", err, wantErr)
			}
		})
	})

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
