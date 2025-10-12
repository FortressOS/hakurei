package container

import (
	"errors"
	"os"
	"syscall"
	"testing"

	"hakurei.app/container/bits"
	"hakurei.app/container/check"
	"hakurei.app/container/stub"
)

func TestBindMountOp(t *testing.T) {
	t.Parallel()

	checkOpBehaviour(t, []opBehaviourTestCase{
		{"ENOENT not optional", new(Params), &BindMountOp{
			Source: check.MustAbs("/bin/"),
			Target: check.MustAbs("/bin/"),
		}, []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/bin/"}, "", syscall.ENOENT),
		}, syscall.ENOENT, nil, nil},

		{"skip optional", new(Params), &BindMountOp{
			Source: check.MustAbs("/bin/"),
			Target: check.MustAbs("/bin/"),
			Flags:  bits.BindOptional,
		}, []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/bin/"}, "", syscall.ENOENT),
		}, nil, nil, nil},

		{"success optional", new(Params), &BindMountOp{
			Source: check.MustAbs("/bin/"),
			Target: check.MustAbs("/bin/"),
			Flags:  bits.BindOptional,
		}, []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/bin/"}, "/usr/bin", nil),
		}, nil, []stub.Call{
			call("stat", stub.ExpectArgs{"/host/usr/bin"}, isDirFi(true), nil),
			call("mkdirAll", stub.ExpectArgs{"/sysroot/bin", os.FileMode(0700)}, nil, nil),
			call("verbosef", stub.ExpectArgs{"mounting %q on %q flags %#x", []any{"/host/usr/bin", "/sysroot/bin", uintptr(0x4005)}}, nil, nil),
			call("bindMount", stub.ExpectArgs{"/host/usr/bin", "/sysroot/bin", uintptr(0x4005), false}, nil, nil),
		}, nil},

		{"ensureFile device", new(Params), &BindMountOp{
			Source: check.MustAbs("/dev/null"),
			Target: check.MustAbs("/dev/null"),
			Flags:  bits.BindWritable | bits.BindDevice,
		}, []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/dev/null"}, "/dev/null", nil),
		}, nil, []stub.Call{
			call("stat", stub.ExpectArgs{"/host/dev/null"}, isDirFi(false), nil),
			call("ensureFile", stub.ExpectArgs{"/sysroot/dev/null", os.FileMode(0444), os.FileMode(0700)}, nil, stub.UniqueError(5)),
		}, stub.UniqueError(5)},

		{"mkdirAll ensure", new(Params), &BindMountOp{
			Source: check.MustAbs("/bin/"),
			Target: check.MustAbs("/bin/"),
			Flags:  bits.BindEnsure,
		}, []stub.Call{
			call("mkdirAll", stub.ExpectArgs{"/bin/", os.FileMode(0700)}, nil, stub.UniqueError(4)),
		}, stub.UniqueError(4), nil, nil},

		{"success ensure", new(Params), &BindMountOp{
			Source: check.MustAbs("/bin/"),
			Target: check.MustAbs("/usr/bin/"),
			Flags:  bits.BindEnsure,
		}, []stub.Call{
			call("mkdirAll", stub.ExpectArgs{"/bin/", os.FileMode(0700)}, nil, nil),
			call("evalSymlinks", stub.ExpectArgs{"/bin/"}, "/usr/bin", nil),
		}, nil, []stub.Call{
			call("stat", stub.ExpectArgs{"/host/usr/bin"}, isDirFi(true), nil),
			call("mkdirAll", stub.ExpectArgs{"/sysroot/usr/bin", os.FileMode(0700)}, nil, nil),
			call("verbosef", stub.ExpectArgs{"mounting %q on %q flags %#x", []any{"/host/usr/bin", "/sysroot/usr/bin", uintptr(0x4005)}}, nil, nil),
			call("bindMount", stub.ExpectArgs{"/host/usr/bin", "/sysroot/usr/bin", uintptr(0x4005), false}, nil, nil),
		}, nil},

		{"success device ro", new(Params), &BindMountOp{
			Source: check.MustAbs("/dev/null"),
			Target: check.MustAbs("/dev/null"),
			Flags:  bits.BindDevice,
		}, []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/dev/null"}, "/dev/null", nil),
		}, nil, []stub.Call{
			call("stat", stub.ExpectArgs{"/host/dev/null"}, isDirFi(false), nil),
			call("ensureFile", stub.ExpectArgs{"/sysroot/dev/null", os.FileMode(0444), os.FileMode(0700)}, nil, nil),
			call("verbosef", stub.ExpectArgs{"mounting %q flags %#x", []any{"/sysroot/dev/null", uintptr(0x4001)}}, nil, nil),
			call("bindMount", stub.ExpectArgs{"/host/dev/null", "/sysroot/dev/null", uintptr(0x4001), false}, nil, nil),
		}, nil},

		{"success device", new(Params), &BindMountOp{
			Source: check.MustAbs("/dev/null"),
			Target: check.MustAbs("/dev/null"),
			Flags:  bits.BindWritable | bits.BindDevice,
		}, []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/dev/null"}, "/dev/null", nil),
		}, nil, []stub.Call{
			call("stat", stub.ExpectArgs{"/host/dev/null"}, isDirFi(false), nil),
			call("ensureFile", stub.ExpectArgs{"/sysroot/dev/null", os.FileMode(0444), os.FileMode(0700)}, nil, nil),
			call("verbosef", stub.ExpectArgs{"mounting %q flags %#x", []any{"/sysroot/dev/null", uintptr(0x4000)}}, nil, nil),
			call("bindMount", stub.ExpectArgs{"/host/dev/null", "/sysroot/dev/null", uintptr(0x4000), false}, nil, nil),
		}, nil},

		{"evalSymlinks", new(Params), &BindMountOp{
			Source: check.MustAbs("/bin/"),
			Target: check.MustAbs("/bin/"),
		}, []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/bin/"}, "/usr/bin", stub.UniqueError(3)),
		}, stub.UniqueError(3), nil, nil},

		{"stat", new(Params), &BindMountOp{
			Source: check.MustAbs("/bin/"),
			Target: check.MustAbs("/bin/"),
		}, []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/bin/"}, "/usr/bin", nil),
		}, nil, []stub.Call{
			call("stat", stub.ExpectArgs{"/host/usr/bin"}, isDirFi(true), stub.UniqueError(2)),
		}, stub.UniqueError(2)},

		{"mkdirAll", new(Params), &BindMountOp{
			Source: check.MustAbs("/bin/"),
			Target: check.MustAbs("/bin/"),
		}, []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/bin/"}, "/usr/bin", nil),
		}, nil, []stub.Call{
			call("stat", stub.ExpectArgs{"/host/usr/bin"}, isDirFi(true), nil),
			call("mkdirAll", stub.ExpectArgs{"/sysroot/bin", os.FileMode(0700)}, nil, stub.UniqueError(1)),
		}, stub.UniqueError(1)},

		{"bindMount", new(Params), &BindMountOp{
			Source: check.MustAbs("/bin/"),
			Target: check.MustAbs("/bin/"),
		}, []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/bin/"}, "/usr/bin", nil),
		}, nil, []stub.Call{
			call("stat", stub.ExpectArgs{"/host/usr/bin"}, isDirFi(true), nil),
			call("mkdirAll", stub.ExpectArgs{"/sysroot/bin", os.FileMode(0700)}, nil, nil),
			call("verbosef", stub.ExpectArgs{"mounting %q on %q flags %#x", []any{"/host/usr/bin", "/sysroot/bin", uintptr(0x4005)}}, nil, nil),
			call("bindMount", stub.ExpectArgs{"/host/usr/bin", "/sysroot/bin", uintptr(0x4005), false}, nil, stub.UniqueError(0)),
		}, stub.UniqueError(0)},

		{"success eval equals", new(Params), &BindMountOp{
			Source: check.MustAbs("/bin/"),
			Target: check.MustAbs("/bin/"),
		}, []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/bin/"}, "/bin", nil),
		}, nil, []stub.Call{
			call("stat", stub.ExpectArgs{"/host/bin"}, isDirFi(true), nil),
			call("mkdirAll", stub.ExpectArgs{"/sysroot/bin", os.FileMode(0700)}, nil, nil),
			call("verbosef", stub.ExpectArgs{"mounting %q on %q flags %#x", []any{"/host/bin", "/sysroot/bin", uintptr(0x4005)}}, nil, nil),
			call("bindMount", stub.ExpectArgs{"/host/bin", "/sysroot/bin", uintptr(0x4005), false}, nil, nil),
		}, nil},

		{"success", new(Params), &BindMountOp{
			Source: check.MustAbs("/bin/"),
			Target: check.MustAbs("/bin/"),
		}, []stub.Call{
			call("evalSymlinks", stub.ExpectArgs{"/bin/"}, "/usr/bin", nil),
		}, nil, []stub.Call{
			call("stat", stub.ExpectArgs{"/host/usr/bin"}, isDirFi(true), nil),
			call("mkdirAll", stub.ExpectArgs{"/sysroot/bin", os.FileMode(0700)}, nil, nil),
			call("verbosef", stub.ExpectArgs{"mounting %q on %q flags %#x", []any{"/host/usr/bin", "/sysroot/bin", uintptr(0x4005)}}, nil, nil),
			call("bindMount", stub.ExpectArgs{"/host/usr/bin", "/sysroot/bin", uintptr(0x4005), false}, nil, nil),
		}, nil},
	})

	t.Run("unreachable", func(t *testing.T) {
		t.Parallel()

		t.Run("nil sourceFinal not optional", func(t *testing.T) {
			t.Parallel()
			wantErr := OpStateError("bind")
			if err := new(BindMountOp).apply(nil, nil); !errors.Is(err, wantErr) {
				t.Errorf("apply: error = %v, want %v", err, wantErr)
			}
		})
	})

	checkOpsValid(t, []opValidTestCase{
		{"nil", (*BindMountOp)(nil), false},
		{"zero", new(BindMountOp), false},
		{"nil source", &BindMountOp{Target: check.MustAbs("/")}, false},
		{"nil target", &BindMountOp{Source: check.MustAbs("/")}, false},
		{"flag optional ensure", &BindMountOp{Source: check.MustAbs("/"), Target: check.MustAbs("/"), Flags: bits.BindOptional | bits.BindEnsure}, false},
		{"valid", &BindMountOp{Source: check.MustAbs("/"), Target: check.MustAbs("/")}, true},
	})

	checkOpsBuilder(t, []opsBuilderTestCase{
		{"autoetc", new(Ops).Bind(
			check.MustAbs("/etc/"),
			check.MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
			0,
		), Ops{
			&BindMountOp{
				Source: check.MustAbs("/etc/"),
				Target: check.MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
			},
		}},
	})

	checkOpIs(t, []opIsTestCase{
		{"zero", new(BindMountOp), new(BindMountOp), false},

		{"internal ne", &BindMountOp{
			Source: check.MustAbs("/etc/"),
			Target: check.MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
		}, &BindMountOp{
			Source:      check.MustAbs("/etc/"),
			Target:      check.MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
			sourceFinal: check.MustAbs("/etc/"),
		}, true},

		{"flags differs", &BindMountOp{
			Source: check.MustAbs("/etc/"),
			Target: check.MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
		}, &BindMountOp{
			Source: check.MustAbs("/etc/"),
			Target: check.MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
			Flags:  bits.BindOptional,
		}, false},

		{"source differs", &BindMountOp{
			Source: check.MustAbs("/.hakurei/etc/"),
			Target: check.MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
		}, &BindMountOp{
			Source: check.MustAbs("/etc/"),
			Target: check.MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
		}, false},

		{"target differs", &BindMountOp{
			Source: check.MustAbs("/etc/"),
			Target: check.MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
		}, &BindMountOp{
			Source: check.MustAbs("/etc/"),
			Target: check.MustAbs("/etc/"),
		}, false},

		{"equals", &BindMountOp{
			Source: check.MustAbs("/etc/"),
			Target: check.MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
		}, &BindMountOp{
			Source: check.MustAbs("/etc/"),
			Target: check.MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
		}, true},
	})

	checkOpMeta(t, []opMetaTestCase{
		{"invalid", new(BindMountOp), "mounting", "<invalid>"},

		{"autoetc", &BindMountOp{
			Source: check.MustAbs("/etc/"),
			Target: check.MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
		}, "mounting", `"/etc/" on "/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659" flags 0x0`},

		{"hostdev", &BindMountOp{
			Source: check.MustAbs("/dev/"),
			Target: check.MustAbs("/dev/"),
			Flags:  bits.BindWritable | bits.BindDevice,
		}, "mounting", `"/dev/" flags 0x6`},
	})
}
