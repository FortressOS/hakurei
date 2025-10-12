package container

import (
	"errors"
	"os"
	"testing"

	"hakurei.app/container/check"
	"hakurei.app/container/stub"
)

func TestMountOverlayOp(t *testing.T) {
	t.Parallel()

	t.Run("argument error", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name string
			err  *OverlayArgumentError
			want string
		}{
			{"unexpected upper", &OverlayArgumentError{OverlayEphemeralUnexpectedUpper, "/proc/"},
				`upperdir has unexpected value "/proc/"`},

			{"lower ro short", &OverlayArgumentError{OverlayReadonlyLower, zeroString},
				"readonly overlay requires at least two lowerdir"},

			{"lower short", &OverlayArgumentError{OverlayEmptyLower, zeroString},
				"overlay requires at least one lowerdir"},

			{"oob", &OverlayArgumentError{0xdeadbeef, zeroString},
				"invalid overlay argument error 0xdeadbeef"},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				if got := tc.err.Error(); got != tc.want {
					t.Errorf("Error: %q, want %q", got, tc.want)
				}
			})
		}
	})

	checkOpBehaviour(t, []opBehaviourTestCase{
		{"mkdirTemp invalid ephemeral", &Params{ParentPerm: 0705}, &MountOverlayOp{
			Target: check.MustAbs("/"),
			Lower: []*check.Absolute{
				check.MustAbs("/var/lib/planterette/base/debian:f92c9052"),
				check.MustAbs("/var/lib/planterette/app/org.chromium.Chromium@debian:f92c9052"),
			},
			Upper: check.MustAbs("/proc/"),
		}, nil, &OverlayArgumentError{OverlayEphemeralUnexpectedUpper, "/proc/"}, nil, nil},

		{"mkdirTemp upper ephemeral", &Params{ParentPerm: 0705}, &MountOverlayOp{
			Target: check.MustAbs("/"),
			Lower: []*check.Absolute{
				check.MustAbs("/var/lib/planterette/base/debian:f92c9052"),
				check.MustAbs("/var/lib/planterette/app/org.chromium.Chromium@debian:f92c9052"),
			},
			Upper: check.MustAbs("/"),
		}, []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/planterette/base/debian:f92c9052"}, "/var/lib/planterette/base/debian:f92c9052", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/planterette/app/org.chromium.Chromium@debian:f92c9052"}, "/var/lib/planterette/app/org.chromium.Chromium@debian:f92c9052", nil),
		}, nil, []stub.Call{
			call("mkdirAll", stub.ExpectArgs{"/sysroot", os.FileMode(0705)}, nil, nil),
			call("mkdirTemp", stub.ExpectArgs{"/", "overlay.upper.*"}, "overlay.upper.32768", stub.UniqueError(6)),
		}, stub.UniqueError(6)},

		{"mkdirTemp work ephemeral", &Params{ParentPerm: 0705}, &MountOverlayOp{
			Target: check.MustAbs("/"),
			Lower: []*check.Absolute{
				check.MustAbs("/var/lib/planterette/base/debian:f92c9052"),
				check.MustAbs("/var/lib/planterette/app/org.chromium.Chromium@debian:f92c9052"),
			},
			Upper: check.MustAbs("/"),
		}, []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/planterette/base/debian:f92c9052"}, "/var/lib/planterette/base/debian:f92c9052", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/planterette/app/org.chromium.Chromium@debian:f92c9052"}, "/var/lib/planterette/app/org.chromium.Chromium@debian:f92c9052", nil),
		}, nil, []stub.Call{
			call("mkdirAll", stub.ExpectArgs{"/sysroot", os.FileMode(0705)}, nil, nil),
			call("mkdirTemp", stub.ExpectArgs{"/", "overlay.upper.*"}, "overlay.upper.32768", nil),
			call("mkdirTemp", stub.ExpectArgs{"/", "overlay.work.*"}, "overlay.work.32768", stub.UniqueError(5)),
		}, stub.UniqueError(5)},

		{"success ephemeral", &Params{ParentPerm: 0705}, &MountOverlayOp{
			Target: check.MustAbs("/"),
			Lower: []*check.Absolute{
				check.MustAbs("/var/lib/planterette/base/debian:f92c9052"),
				check.MustAbs("/var/lib/planterette/app/org.chromium.Chromium@debian:f92c9052"),
			},
			Upper: check.MustAbs("/"),
		}, []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/planterette/base/debian:f92c9052"}, "/var/lib/planterette/base/debian:f92c9052", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/planterette/app/org.chromium.Chromium@debian:f92c9052"}, "/var/lib/planterette/app/org.chromium.Chromium@debian:f92c9052", nil),
		}, nil, []stub.Call{
			call("mkdirAll", stub.ExpectArgs{"/sysroot", os.FileMode(0705)}, nil, nil),
			call("mkdirTemp", stub.ExpectArgs{"/", "overlay.upper.*"}, "overlay.upper.32768", nil),
			call("mkdirTemp", stub.ExpectArgs{"/", "overlay.work.*"}, "overlay.work.32768", nil),
			call("mount", stub.ExpectArgs{"overlay", "/sysroot", "overlay", uintptr(0), "" +
				"upperdir=overlay.upper.32768," +
				"workdir=overlay.work.32768," +
				"lowerdir=" +
				`/host/var/lib/planterette/base/debian\:f92c9052:` +
				`/host/var/lib/planterette/app/org.chromium.Chromium@debian\:f92c9052,` +
				"userxattr"}, nil, nil),
		}, nil},

		{"short lower ro", &Params{ParentPerm: 0755}, &MountOverlayOp{
			Target: check.MustAbs("/nix/store"),
			Lower: []*check.Absolute{
				check.MustAbs("/mnt-root/nix/.ro-store"),
			},
		}, []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/mnt-root/nix/.ro-store"}, "/mnt-root/nix/.ro-store", nil),
		}, nil, []stub.Call{
			call("mkdirAll", stub.ExpectArgs{"/sysroot/nix/store", os.FileMode(0755)}, nil, nil),
		}, &OverlayArgumentError{OverlayReadonlyLower, zeroString}},

		{"success ro noPrefix", &Params{ParentPerm: 0755}, &MountOverlayOp{
			Target: check.MustAbs("/nix/store"),
			Lower: []*check.Absolute{
				check.MustAbs("/mnt-root/nix/.ro-store"),
				check.MustAbs("/mnt-root/nix/.ro-store0"),
			},
			noPrefix: true,
		}, []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/mnt-root/nix/.ro-store"}, "/mnt-root/nix/.ro-store", nil),
			call("evalSymlinks", stub.ExpectArgs{"/mnt-root/nix/.ro-store0"}, "/mnt-root/nix/.ro-store0", nil),
		}, nil, []stub.Call{
			call("mkdirAll", stub.ExpectArgs{"/nix/store", os.FileMode(0755)}, nil, nil),
			call("mount", stub.ExpectArgs{"overlay", "/nix/store", "overlay", uintptr(0), "" +
				"lowerdir=" +
				"/host/mnt-root/nix/.ro-store:" +
				"/host/mnt-root/nix/.ro-store0," +
				"userxattr"}, nil, nil),
		}, nil},

		{"success ro", &Params{ParentPerm: 0755}, &MountOverlayOp{
			Target: check.MustAbs("/nix/store"),
			Lower: []*check.Absolute{
				check.MustAbs("/mnt-root/nix/.ro-store"),
				check.MustAbs("/mnt-root/nix/.ro-store0"),
			},
		}, []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/mnt-root/nix/.ro-store"}, "/mnt-root/nix/.ro-store", nil),
			call("evalSymlinks", stub.ExpectArgs{"/mnt-root/nix/.ro-store0"}, "/mnt-root/nix/.ro-store0", nil),
		}, nil, []stub.Call{
			call("mkdirAll", stub.ExpectArgs{"/sysroot/nix/store", os.FileMode(0755)}, nil, nil),
			call("mount", stub.ExpectArgs{"overlay", "/sysroot/nix/store", "overlay", uintptr(0), "" +
				"lowerdir=" +
				"/host/mnt-root/nix/.ro-store:" +
				"/host/mnt-root/nix/.ro-store0," +
				"userxattr"}, nil, nil),
		}, nil},

		{"nil lower", &Params{ParentPerm: 0700}, &MountOverlayOp{
			Target: check.MustAbs("/nix/store"),
			Upper:  check.MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:   check.MustAbs("/mnt-root/nix/.rw-store/work"),
		}, []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/mnt-root/nix/.rw-store/upper"}, "/mnt-root/nix/.rw-store/.upper", nil),
			call("evalSymlinks", stub.ExpectArgs{"/mnt-root/nix/.rw-store/work"}, "/mnt-root/nix/.rw-store/.work", nil),
		}, nil, []stub.Call{
			call("mkdirAll", stub.ExpectArgs{"/sysroot/nix/store", os.FileMode(0700)}, nil, nil),
		}, &OverlayArgumentError{OverlayEmptyLower, zeroString}},

		{"evalSymlinks upper", &Params{ParentPerm: 0700}, &MountOverlayOp{
			Target: check.MustAbs("/nix/store"),
			Lower:  []*check.Absolute{check.MustAbs("/mnt-root/nix/.ro-store")},
			Upper:  check.MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:   check.MustAbs("/mnt-root/nix/.rw-store/work"),
		}, []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/mnt-root/nix/.rw-store/upper"}, "/mnt-root/nix/.rw-store/.upper", stub.UniqueError(4)),
		}, stub.UniqueError(4), nil, nil},

		{"evalSymlinks work", &Params{ParentPerm: 0700}, &MountOverlayOp{
			Target: check.MustAbs("/nix/store"),
			Lower:  []*check.Absolute{check.MustAbs("/mnt-root/nix/.ro-store")},
			Upper:  check.MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:   check.MustAbs("/mnt-root/nix/.rw-store/work"),
		}, []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/mnt-root/nix/.rw-store/upper"}, "/mnt-root/nix/.rw-store/.upper", nil),
			call("evalSymlinks", stub.ExpectArgs{"/mnt-root/nix/.rw-store/work"}, "/mnt-root/nix/.rw-store/.work", stub.UniqueError(3)),
		}, stub.UniqueError(3), nil, nil},

		{"evalSymlinks lower", &Params{ParentPerm: 0700}, &MountOverlayOp{
			Target: check.MustAbs("/nix/store"),
			Lower:  []*check.Absolute{check.MustAbs("/mnt-root/nix/.ro-store")},
			Upper:  check.MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:   check.MustAbs("/mnt-root/nix/.rw-store/work"),
		}, []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/mnt-root/nix/.rw-store/upper"}, "/mnt-root/nix/.rw-store/.upper", nil),
			call("evalSymlinks", stub.ExpectArgs{"/mnt-root/nix/.rw-store/work"}, "/mnt-root/nix/.rw-store/.work", nil),
			call("evalSymlinks", stub.ExpectArgs{"/mnt-root/nix/.ro-store"}, "/mnt-root/nix/ro-store", stub.UniqueError(2)),
		}, stub.UniqueError(2), nil, nil},

		{"mkdirAll", &Params{ParentPerm: 0700}, &MountOverlayOp{
			Target: check.MustAbs("/nix/store"),
			Lower:  []*check.Absolute{check.MustAbs("/mnt-root/nix/.ro-store")},
			Upper:  check.MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:   check.MustAbs("/mnt-root/nix/.rw-store/work"),
		}, []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/mnt-root/nix/.rw-store/upper"}, "/mnt-root/nix/.rw-store/.upper", nil),
			call("evalSymlinks", stub.ExpectArgs{"/mnt-root/nix/.rw-store/work"}, "/mnt-root/nix/.rw-store/.work", nil),
			call("evalSymlinks", stub.ExpectArgs{"/mnt-root/nix/.ro-store"}, "/mnt-root/nix/ro-store", nil),
		}, nil, []stub.Call{
			call("mkdirAll", stub.ExpectArgs{"/sysroot/nix/store", os.FileMode(0700)}, nil, stub.UniqueError(1)),
		}, stub.UniqueError(1)},

		{"mount", &Params{ParentPerm: 0700}, &MountOverlayOp{
			Target: check.MustAbs("/nix/store"),
			Lower:  []*check.Absolute{check.MustAbs("/mnt-root/nix/.ro-store")},
			Upper:  check.MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:   check.MustAbs("/mnt-root/nix/.rw-store/work"),
		}, []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/mnt-root/nix/.rw-store/upper"}, "/mnt-root/nix/.rw-store/.upper", nil),
			call("evalSymlinks", stub.ExpectArgs{"/mnt-root/nix/.rw-store/work"}, "/mnt-root/nix/.rw-store/.work", nil),
			call("evalSymlinks", stub.ExpectArgs{"/mnt-root/nix/.ro-store"}, "/mnt-root/nix/ro-store", nil),
		}, nil, []stub.Call{
			call("mkdirAll", stub.ExpectArgs{"/sysroot/nix/store", os.FileMode(0700)}, nil, nil),
			call("mount", stub.ExpectArgs{"overlay", "/sysroot/nix/store", "overlay", uintptr(0), "upperdir=/host/mnt-root/nix/.rw-store/.upper,workdir=/host/mnt-root/nix/.rw-store/.work,lowerdir=/host/mnt-root/nix/ro-store,userxattr"}, nil, stub.UniqueError(0)),
		}, stub.UniqueError(0)},

		{"success single layer", &Params{ParentPerm: 0700}, &MountOverlayOp{
			Target: check.MustAbs("/nix/store"),
			Lower:  []*check.Absolute{check.MustAbs("/mnt-root/nix/.ro-store")},
			Upper:  check.MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:   check.MustAbs("/mnt-root/nix/.rw-store/work"),
		}, []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/mnt-root/nix/.rw-store/upper"}, "/mnt-root/nix/.rw-store/.upper", nil),
			call("evalSymlinks", stub.ExpectArgs{"/mnt-root/nix/.rw-store/work"}, "/mnt-root/nix/.rw-store/.work", nil),
			call("evalSymlinks", stub.ExpectArgs{"/mnt-root/nix/.ro-store"}, "/mnt-root/nix/ro-store", nil),
		}, nil, []stub.Call{
			call("mkdirAll", stub.ExpectArgs{"/sysroot/nix/store", os.FileMode(0700)}, nil, nil),
			call("mount", stub.ExpectArgs{"overlay", "/sysroot/nix/store", "overlay", uintptr(0), "" +
				"upperdir=/host/mnt-root/nix/.rw-store/.upper," +
				"workdir=/host/mnt-root/nix/.rw-store/.work," +
				"lowerdir=/host/mnt-root/nix/ro-store," +
				"userxattr"}, nil, nil),
		}, nil},

		{"success", &Params{ParentPerm: 0700}, &MountOverlayOp{
			Target: check.MustAbs("/nix/store"),
			Lower: []*check.Absolute{
				check.MustAbs("/mnt-root/nix/.ro-store"),
				check.MustAbs("/mnt-root/nix/.ro-store0"),
				check.MustAbs("/mnt-root/nix/.ro-store1"),
				check.MustAbs("/mnt-root/nix/.ro-store2"),
				check.MustAbs("/mnt-root/nix/.ro-store3"),
			},
			Upper: check.MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:  check.MustAbs("/mnt-root/nix/.rw-store/work"),
		}, []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/mnt-root/nix/.rw-store/upper"}, "/mnt-root/nix/.rw-store/.upper", nil),
			call("evalSymlinks", stub.ExpectArgs{"/mnt-root/nix/.rw-store/work"}, "/mnt-root/nix/.rw-store/.work", nil),
			call("evalSymlinks", stub.ExpectArgs{"/mnt-root/nix/.ro-store"}, "/mnt-root/nix/ro-store", nil),
			call("evalSymlinks", stub.ExpectArgs{"/mnt-root/nix/.ro-store0"}, "/mnt-root/nix/ro-store0", nil),
			call("evalSymlinks", stub.ExpectArgs{"/mnt-root/nix/.ro-store1"}, "/mnt-root/nix/ro-store1", nil),
			call("evalSymlinks", stub.ExpectArgs{"/mnt-root/nix/.ro-store2"}, "/mnt-root/nix/ro-store2", nil),
			call("evalSymlinks", stub.ExpectArgs{"/mnt-root/nix/.ro-store3"}, "/mnt-root/nix/ro-store3", nil),
		}, nil, []stub.Call{
			call("mkdirAll", stub.ExpectArgs{"/sysroot/nix/store", os.FileMode(0700)}, nil, nil),
			call("mount", stub.ExpectArgs{"overlay", "/sysroot/nix/store", "overlay", uintptr(0), "" +
				"upperdir=/host/mnt-root/nix/.rw-store/.upper," +
				"workdir=/host/mnt-root/nix/.rw-store/.work," +
				"lowerdir=" +
				"/host/mnt-root/nix/ro-store:" +
				"/host/mnt-root/nix/ro-store0:" +
				"/host/mnt-root/nix/ro-store1:" +
				"/host/mnt-root/nix/ro-store2:" +
				"/host/mnt-root/nix/ro-store3," +
				"userxattr"}, nil, nil),
		}, nil},
	})

	t.Run("unreachable", func(t *testing.T) {
		t.Parallel()

		t.Run("nil Upper non-nil Work not ephemeral", func(t *testing.T) {
			t.Parallel()
			wantErr := OpStateError("overlay")
			if err := (&MountOverlayOp{
				Work: check.MustAbs("/"),
			}).early(nil, nil); !errors.Is(err, wantErr) {
				t.Errorf("apply: error = %v, want %v", err, wantErr)
			}
		})
	})

	checkOpsValid(t, []opValidTestCase{
		{"nil", (*MountOverlayOp)(nil), false},
		{"zero", new(MountOverlayOp), false},
		{"nil lower", &MountOverlayOp{Target: check.MustAbs("/"), Lower: []*check.Absolute{nil}}, false},
		{"ro", &MountOverlayOp{Target: check.MustAbs("/"), Lower: []*check.Absolute{check.MustAbs("/")}}, true},
		{"ro work", &MountOverlayOp{Target: check.MustAbs("/"), Work: check.MustAbs("/tmp/")}, false},
		{"rw", &MountOverlayOp{Target: check.MustAbs("/"), Lower: []*check.Absolute{check.MustAbs("/")}, Upper: check.MustAbs("/"), Work: check.MustAbs("/")}, true},
	})

	checkOpsBuilder(t, []opsBuilderTestCase{
		{"full", new(Ops).Overlay(
			check.MustAbs("/nix/store"),
			check.MustAbs("/mnt-root/nix/.rw-store/upper"),
			check.MustAbs("/mnt-root/nix/.rw-store/work"),
			check.MustAbs("/mnt-root/nix/.ro-store"),
		), Ops{
			&MountOverlayOp{
				Target: check.MustAbs("/nix/store"),
				Lower:  []*check.Absolute{check.MustAbs("/mnt-root/nix/.ro-store")},
				Upper:  check.MustAbs("/mnt-root/nix/.rw-store/upper"),
				Work:   check.MustAbs("/mnt-root/nix/.rw-store/work"),
			},
		}},

		{"ephemeral", new(Ops).OverlayEphemeral(check.MustAbs("/nix/store"), check.MustAbs("/mnt-root/nix/.ro-store")), Ops{
			&MountOverlayOp{
				Target: check.MustAbs("/nix/store"),
				Lower:  []*check.Absolute{check.MustAbs("/mnt-root/nix/.ro-store")},
				Upper:  check.MustAbs("/"),
			},
		}},

		{"readonly", new(Ops).OverlayReadonly(check.MustAbs("/nix/store"), check.MustAbs("/mnt-root/nix/.ro-store")), Ops{
			&MountOverlayOp{
				Target: check.MustAbs("/nix/store"),
				Lower:  []*check.Absolute{check.MustAbs("/mnt-root/nix/.ro-store")},
			},
		}},
	})

	checkOpIs(t, []opIsTestCase{
		{"zero", new(MountOverlayOp), new(MountOverlayOp), false},

		{"differs target", &MountOverlayOp{
			Target: check.MustAbs("/nix/store/differs"),
			Lower:  []*check.Absolute{check.MustAbs("/mnt-root/nix/.ro-store")},
			Upper:  check.MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:   check.MustAbs("/mnt-root/nix/.rw-store/work"),
		}, &MountOverlayOp{
			Target: check.MustAbs("/nix/store"),
			Lower:  []*check.Absolute{check.MustAbs("/mnt-root/nix/.ro-store")},
			Upper:  check.MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:   check.MustAbs("/mnt-root/nix/.rw-store/work")}, false},

		{"differs lower", &MountOverlayOp{
			Target: check.MustAbs("/nix/store"),
			Lower:  []*check.Absolute{check.MustAbs("/mnt-root/nix/.ro-store/differs")},
			Upper:  check.MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:   check.MustAbs("/mnt-root/nix/.rw-store/work"),
		}, &MountOverlayOp{
			Target: check.MustAbs("/nix/store"),
			Lower:  []*check.Absolute{check.MustAbs("/mnt-root/nix/.ro-store")},
			Upper:  check.MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:   check.MustAbs("/mnt-root/nix/.rw-store/work")}, false},

		{"differs upper", &MountOverlayOp{
			Target: check.MustAbs("/nix/store"),
			Lower:  []*check.Absolute{check.MustAbs("/mnt-root/nix/.ro-store")},
			Upper:  check.MustAbs("/mnt-root/nix/.rw-store/upper/differs"),
			Work:   check.MustAbs("/mnt-root/nix/.rw-store/work"),
		}, &MountOverlayOp{
			Target: check.MustAbs("/nix/store"),
			Lower:  []*check.Absolute{check.MustAbs("/mnt-root/nix/.ro-store")},
			Upper:  check.MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:   check.MustAbs("/mnt-root/nix/.rw-store/work")}, false},

		{"differs work", &MountOverlayOp{
			Target: check.MustAbs("/nix/store"),
			Lower:  []*check.Absolute{check.MustAbs("/mnt-root/nix/.ro-store")},
			Upper:  check.MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:   check.MustAbs("/mnt-root/nix/.rw-store/work/differs"),
		}, &MountOverlayOp{
			Target: check.MustAbs("/nix/store"),
			Lower:  []*check.Absolute{check.MustAbs("/mnt-root/nix/.ro-store")},
			Upper:  check.MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:   check.MustAbs("/mnt-root/nix/.rw-store/work")}, false},

		{"equals ro", &MountOverlayOp{
			Target: check.MustAbs("/nix/store"),
			Lower:  []*check.Absolute{check.MustAbs("/mnt-root/nix/.ro-store")},
		}, &MountOverlayOp{
			Target: check.MustAbs("/nix/store"),
			Lower:  []*check.Absolute{check.MustAbs("/mnt-root/nix/.ro-store")}}, true},

		{"equals", &MountOverlayOp{
			Target: check.MustAbs("/nix/store"),
			Lower:  []*check.Absolute{check.MustAbs("/mnt-root/nix/.ro-store")},
			Upper:  check.MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:   check.MustAbs("/mnt-root/nix/.rw-store/work"),
		}, &MountOverlayOp{
			Target: check.MustAbs("/nix/store"),
			Lower:  []*check.Absolute{check.MustAbs("/mnt-root/nix/.ro-store")},
			Upper:  check.MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:   check.MustAbs("/mnt-root/nix/.rw-store/work")}, true},
	})

	checkOpMeta(t, []opMetaTestCase{
		{"nix", &MountOverlayOp{
			Target: check.MustAbs("/nix/store"),
			Lower:  []*check.Absolute{check.MustAbs("/mnt-root/nix/.ro-store")},
			Upper:  check.MustAbs("/mnt-root/nix/.rw-store/upper"),
			Work:   check.MustAbs("/mnt-root/nix/.rw-store/work"),
		}, "mounting", `overlay on "/nix/store" with 1 layers`},
	})
}
