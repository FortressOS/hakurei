package container

import (
	"errors"
	"os"
	"syscall"
	"testing"
)

func TestBindMountOp(t *testing.T) {
	checkOpBehaviour(t, []opBehaviourTestCase{
		{"ENOENT not optional", new(Params), &BindMountOp{
			Source: MustAbs("/bin/"),
			Target: MustAbs("/bin/"),
		}, []kexpect{
			{"evalSymlinks", expectArgs{"/bin/"}, "", syscall.ENOENT},
		}, wrapErrSelf(syscall.ENOENT), nil, nil},

		{"skip optional", new(Params), &BindMountOp{
			Source: MustAbs("/bin/"),
			Target: MustAbs("/bin/"),
			Flags:  BindOptional,
		}, []kexpect{
			{"evalSymlinks", expectArgs{"/bin/"}, "", syscall.ENOENT},
		}, nil, nil, nil},

		{"success optional", new(Params), &BindMountOp{
			Source: MustAbs("/bin/"),
			Target: MustAbs("/bin/"),
			Flags:  BindOptional,
		}, []kexpect{
			{"evalSymlinks", expectArgs{"/bin/"}, "/usr/bin", nil},
		}, nil, []kexpect{
			{"stat", expectArgs{"/host/usr/bin"}, isDirFi(true), nil},
			{"mkdirAll", expectArgs{"/sysroot/bin", os.FileMode(0700)}, nil, nil},
			{"bindMount", expectArgs{"/host/usr/bin", "/sysroot/bin", uintptr(0x4005), false}, nil, nil},
		}, nil},

		{"ensureFile device", new(Params), &BindMountOp{
			Source: MustAbs("/dev/null"),
			Target: MustAbs("/dev/null"),
			Flags:  BindWritable | BindDevice,
		}, []kexpect{
			{"evalSymlinks", expectArgs{"/dev/null"}, "/dev/null", nil},
		}, nil, []kexpect{
			{"stat", expectArgs{"/host/dev/null"}, isDirFi(false), nil},
			{"ensureFile", expectArgs{"/sysroot/dev/null", os.FileMode(0444), os.FileMode(0700)}, nil, errUnique},
		}, errUnique},

		{"mkdirAll ensure", new(Params), &BindMountOp{
			Source: MustAbs("/bin/"),
			Target: MustAbs("/bin/"),
			Flags:  BindEnsure,
		}, []kexpect{
			{"mkdirAll", expectArgs{"/bin/", os.FileMode(0700)}, nil, errUnique},
		}, wrapErrSelf(errUnique), nil, nil},

		{"success ensure", new(Params), &BindMountOp{
			Source: MustAbs("/bin/"),
			Target: MustAbs("/usr/bin/"),
			Flags:  BindEnsure,
		}, []kexpect{
			{"mkdirAll", expectArgs{"/bin/", os.FileMode(0700)}, nil, nil},
			{"evalSymlinks", expectArgs{"/bin/"}, "/usr/bin", nil},
		}, nil, []kexpect{
			{"stat", expectArgs{"/host/usr/bin"}, isDirFi(true), nil},
			{"mkdirAll", expectArgs{"/sysroot/usr/bin", os.FileMode(0700)}, nil, nil},
			{"bindMount", expectArgs{"/host/usr/bin", "/sysroot/usr/bin", uintptr(0x4005), false}, nil, nil},
		}, nil},

		{"success device ro", new(Params), &BindMountOp{
			Source: MustAbs("/dev/null"),
			Target: MustAbs("/dev/null"),
			Flags:  BindDevice,
		}, []kexpect{
			{"evalSymlinks", expectArgs{"/dev/null"}, "/dev/null", nil},
		}, nil, []kexpect{
			{"stat", expectArgs{"/host/dev/null"}, isDirFi(false), nil},
			{"ensureFile", expectArgs{"/sysroot/dev/null", os.FileMode(0444), os.FileMode(0700)}, nil, nil},
			{"bindMount", expectArgs{"/host/dev/null", "/sysroot/dev/null", uintptr(0x4001), false}, nil, nil},
		}, nil},

		{"success device", new(Params), &BindMountOp{
			Source: MustAbs("/dev/null"),
			Target: MustAbs("/dev/null"),
			Flags:  BindWritable | BindDevice,
		}, []kexpect{
			{"evalSymlinks", expectArgs{"/dev/null"}, "/dev/null", nil},
		}, nil, []kexpect{
			{"stat", expectArgs{"/host/dev/null"}, isDirFi(false), nil},
			{"ensureFile", expectArgs{"/sysroot/dev/null", os.FileMode(0444), os.FileMode(0700)}, nil, nil},
			{"bindMount", expectArgs{"/host/dev/null", "/sysroot/dev/null", uintptr(0x4000), false}, nil, nil},
		}, nil},

		{"evalSymlinks", new(Params), &BindMountOp{
			Source: MustAbs("/bin/"),
			Target: MustAbs("/bin/"),
		}, []kexpect{
			{"evalSymlinks", expectArgs{"/bin/"}, "/usr/bin", errUnique},
		}, wrapErrSelf(errUnique), nil, nil},

		{"stat", new(Params), &BindMountOp{
			Source: MustAbs("/bin/"),
			Target: MustAbs("/bin/"),
		}, []kexpect{
			{"evalSymlinks", expectArgs{"/bin/"}, "/usr/bin", nil},
		}, nil, []kexpect{
			{"stat", expectArgs{"/host/usr/bin"}, isDirFi(true), errUnique},
		}, wrapErrSelf(errUnique)},

		{"mkdirAll", new(Params), &BindMountOp{
			Source: MustAbs("/bin/"),
			Target: MustAbs("/bin/"),
		}, []kexpect{
			{"evalSymlinks", expectArgs{"/bin/"}, "/usr/bin", nil},
		}, nil, []kexpect{
			{"stat", expectArgs{"/host/usr/bin"}, isDirFi(true), nil},
			{"mkdirAll", expectArgs{"/sysroot/bin", os.FileMode(0700)}, nil, errUnique},
		}, wrapErrSelf(errUnique)},

		{"bindMount", new(Params), &BindMountOp{
			Source: MustAbs("/bin/"),
			Target: MustAbs("/bin/"),
		}, []kexpect{
			{"evalSymlinks", expectArgs{"/bin/"}, "/usr/bin", nil},
		}, nil, []kexpect{
			{"stat", expectArgs{"/host/usr/bin"}, isDirFi(true), nil},
			{"mkdirAll", expectArgs{"/sysroot/bin", os.FileMode(0700)}, nil, nil},
			{"bindMount", expectArgs{"/host/usr/bin", "/sysroot/bin", uintptr(0x4005), false}, nil, errUnique},
		}, errUnique},

		{"success", new(Params), &BindMountOp{
			Source: MustAbs("/bin/"),
			Target: MustAbs("/bin/"),
		}, []kexpect{
			{"evalSymlinks", expectArgs{"/bin/"}, "/usr/bin", nil},
		}, nil, []kexpect{
			{"stat", expectArgs{"/host/usr/bin"}, isDirFi(true), nil},
			{"mkdirAll", expectArgs{"/sysroot/bin", os.FileMode(0700)}, nil, nil},
			{"bindMount", expectArgs{"/host/usr/bin", "/sysroot/bin", uintptr(0x4005), false}, nil, nil},
		}, nil},
	})

	t.Run("unreachable", func(t *testing.T) {
		t.Run("nil sourceFinal not optional", func(t *testing.T) {
			wantErr := msg.WrapErr(os.ErrClosed, "impossible bind state reached")
			if err := new(BindMountOp).apply(nil, nil); !errors.Is(err, wantErr) {
				t.Errorf("apply: error = %v, want %v", err, wantErr)
			}
		})
	})

	checkOpsValid(t, []opValidTestCase{
		{"nil", (*BindMountOp)(nil), false},
		{"zero", new(BindMountOp), false},
		{"nil source", &BindMountOp{Target: MustAbs("/")}, false},
		{"nil target", &BindMountOp{Source: MustAbs("/")}, false},
		{"flag optional ensure", &BindMountOp{Source: MustAbs("/"), Target: MustAbs("/"), Flags: BindOptional | BindEnsure}, false},
		{"valid", &BindMountOp{Source: MustAbs("/"), Target: MustAbs("/")}, true},
	})

	checkOpsBuilder(t, []opsBuilderTestCase{
		{"autoetc", new(Ops).Bind(
			MustAbs("/etc/"),
			MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
			0,
		), Ops{
			&BindMountOp{
				Source: MustAbs("/etc/"),
				Target: MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
			},
		}},
	})

	checkOpIs(t, []opIsTestCase{
		{"zero", new(BindMountOp), new(BindMountOp), false},

		{"internal ne", &BindMountOp{
			Source: MustAbs("/etc/"),
			Target: MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
		}, &BindMountOp{
			Source:      MustAbs("/etc/"),
			Target:      MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
			sourceFinal: MustAbs("/etc/"),
		}, true},

		{"flags differs", &BindMountOp{
			Source: MustAbs("/etc/"),
			Target: MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
		}, &BindMountOp{
			Source: MustAbs("/etc/"),
			Target: MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
			Flags:  BindOptional,
		}, false},

		{"source differs", &BindMountOp{
			Source: MustAbs("/.hakurei/etc/"),
			Target: MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
		}, &BindMountOp{
			Source: MustAbs("/etc/"),
			Target: MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
		}, false},

		{"target differs", &BindMountOp{
			Source: MustAbs("/etc/"),
			Target: MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
		}, &BindMountOp{
			Source: MustAbs("/etc/"),
			Target: MustAbs("/etc/"),
		}, false},

		{"equals", &BindMountOp{
			Source: MustAbs("/etc/"),
			Target: MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
		}, &BindMountOp{
			Source: MustAbs("/etc/"),
			Target: MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
		}, true},
	})

	checkOpMeta(t, []opMetaTestCase{
		{"invalid", new(BindMountOp), "mounting", "<invalid>"},

		{"autoetc", &BindMountOp{
			Source: MustAbs("/etc/"),
			Target: MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
		}, "mounting", `"/etc/" on "/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659" flags 0x0`},

		{"hostdev", &BindMountOp{
			Source: MustAbs("/dev/"),
			Target: MustAbs("/dev/"),
			Flags:  BindWritable | BindDevice,
		}, "mounting", `"/dev/" flags 0x6`},
	})
}
