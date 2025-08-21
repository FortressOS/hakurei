package container

import (
	"errors"
	"io/fs"
	"os"
	"testing"
)

func TestAutoRootOp(t *testing.T) {
	t.Run("nonrepeatable", func(t *testing.T) {
		wantErr := msg.WrapErr(fs.ErrInvalid, "autoroot is not repeatable")
		if err := (&AutoRootOp{Prefix: "81ceabb30d37bbdb3868004629cb84e9"}).apply(&setupState{nonrepeatable: nrAutoRoot}, nil); !errors.Is(err, wantErr) {
			t.Errorf("apply: error = %v, want %v", err, wantErr)
		}
	})

	checkOpBehaviour(t, []opBehaviourTestCase{
		{"readdir", &Params{ParentPerm: 0750}, &AutoRootOp{
			Host:   MustAbs("/"),
			Prefix: "81ceabb30d37bbdb3868004629cb84e9",
			Flags:  BindWritable,
		}, []kexpect{
			{"readdir", expectArgs{"/"}, stubDir(), errUnique},
		}, wrapErrSelf(errUnique), nil, nil},

		{"early", &Params{ParentPerm: 0750}, &AutoRootOp{
			Host:   MustAbs("/"),
			Prefix: "81ceabb30d37bbdb3868004629cb84e9",
			Flags:  BindWritable,
		}, []kexpect{
			{"readdir", expectArgs{"/"}, stubDir("bin", "dev", "etc", "home", "lib64",
				"lost+found", "mnt", "nix", "proc", "root", "run", "srv", "sys", "tmp", "usr", "var"), nil},
			{"evalSymlinks", expectArgs{"/bin"}, "", errUnique},
		}, wrapErrSelf(errUnique), nil, nil},

		{"apply", &Params{ParentPerm: 0750}, &AutoRootOp{
			Host:   MustAbs("/"),
			Prefix: "81ceabb30d37bbdb3868004629cb84e9",
			Flags:  BindWritable,
		}, []kexpect{
			{"readdir", expectArgs{"/"}, stubDir("bin", "dev", "etc", "home", "lib64",
				"lost+found", "mnt", "nix", "proc", "root", "run", "srv", "sys", "tmp", "usr", "var"), nil},
			{"evalSymlinks", expectArgs{"/bin"}, "/usr/bin", nil},
			{"evalSymlinks", expectArgs{"/home"}, "/home", nil},
			{"evalSymlinks", expectArgs{"/lib64"}, "/lib64", nil},
			{"evalSymlinks", expectArgs{"/lost+found"}, "/lost+found", nil},
			{"evalSymlinks", expectArgs{"/nix"}, "/nix", nil},
			{"evalSymlinks", expectArgs{"/root"}, "/root", nil},
			{"evalSymlinks", expectArgs{"/run"}, "/run", nil},
			{"evalSymlinks", expectArgs{"/srv"}, "/srv", nil},
			{"evalSymlinks", expectArgs{"/sys"}, "/sys", nil},
			{"evalSymlinks", expectArgs{"/usr"}, "/usr", nil},
			{"evalSymlinks", expectArgs{"/var"}, "/var", nil},
		}, nil, []kexpect{
			{"verbosef", expectArgs{"%s %s", []any{"mounting", &BindMountOp{MustAbs("/usr/bin"), MustAbs("/bin"), MustAbs("/bin"), BindWritable}}}, nil, nil},
			{"stat", expectArgs{"/host/usr/bin"}, isDirFi(false), errUnique},
		}, wrapErrSelf(errUnique)},

		{"success pd", &Params{ParentPerm: 0750}, &AutoRootOp{
			Host:   MustAbs("/"),
			Prefix: "81ceabb30d37bbdb3868004629cb84e9",
			Flags:  BindWritable,
		}, []kexpect{
			{"readdir", expectArgs{"/"}, stubDir("bin", "dev", "etc", "home", "lib64",
				"lost+found", "mnt", "nix", "proc", "root", "run", "srv", "sys", "tmp", "usr", "var"), nil},
			{"evalSymlinks", expectArgs{"/bin"}, "/usr/bin", nil},
			{"evalSymlinks", expectArgs{"/home"}, "/home", nil},
			{"evalSymlinks", expectArgs{"/lib64"}, "/lib64", nil},
			{"evalSymlinks", expectArgs{"/lost+found"}, "/lost+found", nil},
			{"evalSymlinks", expectArgs{"/nix"}, "/nix", nil},
			{"evalSymlinks", expectArgs{"/root"}, "/root", nil},
			{"evalSymlinks", expectArgs{"/run"}, "/run", nil},
			{"evalSymlinks", expectArgs{"/srv"}, "/srv", nil},
			{"evalSymlinks", expectArgs{"/sys"}, "/sys", nil},
			{"evalSymlinks", expectArgs{"/usr"}, "/usr", nil},
			{"evalSymlinks", expectArgs{"/var"}, "/var", nil},
		}, nil, []kexpect{
			{"verbosef", expectArgs{"%s %s", []any{"mounting", &BindMountOp{MustAbs("/usr/bin"), MustAbs("/bin"), MustAbs("/bin"), BindWritable}}}, nil, nil}, {"stat", expectArgs{"/host/usr/bin"}, isDirFi(true), nil}, {"mkdirAll", expectArgs{"/sysroot/bin", os.FileMode(0700)}, nil, nil}, {"bindMount", expectArgs{"/host/usr/bin", "/sysroot/bin", uintptr(0x4004), false}, nil, nil},
			{"verbosef", expectArgs{"%s %s", []any{"mounting", &BindMountOp{MustAbs("/home"), MustAbs("/home"), MustAbs("/home"), BindWritable}}}, nil, nil}, {"stat", expectArgs{"/host/home"}, isDirFi(true), nil}, {"mkdirAll", expectArgs{"/sysroot/home", os.FileMode(0700)}, nil, nil}, {"bindMount", expectArgs{"/host/home", "/sysroot/home", uintptr(0x4004), false}, nil, nil},
			{"verbosef", expectArgs{"%s %s", []any{"mounting", &BindMountOp{MustAbs("/lib64"), MustAbs("/lib64"), MustAbs("/lib64"), BindWritable}}}, nil, nil}, {"stat", expectArgs{"/host/lib64"}, isDirFi(true), nil}, {"mkdirAll", expectArgs{"/sysroot/lib64", os.FileMode(0700)}, nil, nil}, {"bindMount", expectArgs{"/host/lib64", "/sysroot/lib64", uintptr(0x4004), false}, nil, nil},
			{"verbosef", expectArgs{"%s %s", []any{"mounting", &BindMountOp{MustAbs("/lost+found"), MustAbs("/lost+found"), MustAbs("/lost+found"), BindWritable}}}, nil, nil}, {"stat", expectArgs{"/host/lost+found"}, isDirFi(true), nil}, {"mkdirAll", expectArgs{"/sysroot/lost+found", os.FileMode(0700)}, nil, nil}, {"bindMount", expectArgs{"/host/lost+found", "/sysroot/lost+found", uintptr(0x4004), false}, nil, nil},
			{"verbosef", expectArgs{"%s %s", []any{"mounting", &BindMountOp{MustAbs("/nix"), MustAbs("/nix"), MustAbs("/nix"), BindWritable}}}, nil, nil}, {"stat", expectArgs{"/host/nix"}, isDirFi(true), nil}, {"mkdirAll", expectArgs{"/sysroot/nix", os.FileMode(0700)}, nil, nil}, {"bindMount", expectArgs{"/host/nix", "/sysroot/nix", uintptr(0x4004), false}, nil, nil},
			{"verbosef", expectArgs{"%s %s", []any{"mounting", &BindMountOp{MustAbs("/root"), MustAbs("/root"), MustAbs("/root"), BindWritable}}}, nil, nil}, {"stat", expectArgs{"/host/root"}, isDirFi(true), nil}, {"mkdirAll", expectArgs{"/sysroot/root", os.FileMode(0700)}, nil, nil}, {"bindMount", expectArgs{"/host/root", "/sysroot/root", uintptr(0x4004), false}, nil, nil},
			{"verbosef", expectArgs{"%s %s", []any{"mounting", &BindMountOp{MustAbs("/run"), MustAbs("/run"), MustAbs("/run"), BindWritable}}}, nil, nil}, {"stat", expectArgs{"/host/run"}, isDirFi(true), nil}, {"mkdirAll", expectArgs{"/sysroot/run", os.FileMode(0700)}, nil, nil}, {"bindMount", expectArgs{"/host/run", "/sysroot/run", uintptr(0x4004), false}, nil, nil},
			{"verbosef", expectArgs{"%s %s", []any{"mounting", &BindMountOp{MustAbs("/srv"), MustAbs("/srv"), MustAbs("/srv"), BindWritable}}}, nil, nil}, {"stat", expectArgs{"/host/srv"}, isDirFi(true), nil}, {"mkdirAll", expectArgs{"/sysroot/srv", os.FileMode(0700)}, nil, nil}, {"bindMount", expectArgs{"/host/srv", "/sysroot/srv", uintptr(0x4004), false}, nil, nil},
			{"verbosef", expectArgs{"%s %s", []any{"mounting", &BindMountOp{MustAbs("/sys"), MustAbs("/sys"), MustAbs("/sys"), BindWritable}}}, nil, nil}, {"stat", expectArgs{"/host/sys"}, isDirFi(true), nil}, {"mkdirAll", expectArgs{"/sysroot/sys", os.FileMode(0700)}, nil, nil}, {"bindMount", expectArgs{"/host/sys", "/sysroot/sys", uintptr(0x4004), false}, nil, nil},
			{"verbosef", expectArgs{"%s %s", []any{"mounting", &BindMountOp{MustAbs("/usr"), MustAbs("/usr"), MustAbs("/usr"), BindWritable}}}, nil, nil}, {"stat", expectArgs{"/host/usr"}, isDirFi(true), nil}, {"mkdirAll", expectArgs{"/sysroot/usr", os.FileMode(0700)}, nil, nil}, {"bindMount", expectArgs{"/host/usr", "/sysroot/usr", uintptr(0x4004), false}, nil, nil},
			{"verbosef", expectArgs{"%s %s", []any{"mounting", &BindMountOp{MustAbs("/var"), MustAbs("/var"), MustAbs("/var"), BindWritable}}}, nil, nil}, {"stat", expectArgs{"/host/var"}, isDirFi(true), nil}, {"mkdirAll", expectArgs{"/sysroot/var", os.FileMode(0700)}, nil, nil}, {"bindMount", expectArgs{"/host/var", "/sysroot/var", uintptr(0x4004), false}, nil, nil},
		}, nil},

		{"success", &Params{ParentPerm: 0750}, &AutoRootOp{
			Host:   MustAbs("/var/lib/planterette/base/debian:f92c9052"),
			Prefix: "81ceabb30d37bbdb3868004629cb84e9",
		}, []kexpect{
			{"readdir", expectArgs{"/var/lib/planterette/base/debian:f92c9052"}, stubDir("bin", "dev", "etc", "home", "lib64",
				"lost+found", "mnt", "nix", "proc", "root", "run", "srv", "sys", "tmp", "usr", "var"), nil},
			{"evalSymlinks", expectArgs{"/var/lib/planterette/base/debian:f92c9052/bin"}, "/var/lib/planterette/base/debian:f92c9052/usr/bin", nil},
			{"evalSymlinks", expectArgs{"/var/lib/planterette/base/debian:f92c9052/home"}, "/var/lib/planterette/base/debian:f92c9052/home", nil},
			{"evalSymlinks", expectArgs{"/var/lib/planterette/base/debian:f92c9052/lib64"}, "/var/lib/planterette/base/debian:f92c9052/lib64", nil},
			{"evalSymlinks", expectArgs{"/var/lib/planterette/base/debian:f92c9052/lost+found"}, "/var/lib/planterette/base/debian:f92c9052/lost+found", nil},
			{"evalSymlinks", expectArgs{"/var/lib/planterette/base/debian:f92c9052/nix"}, "/var/lib/planterette/base/debian:f92c9052/nix", nil},
			{"evalSymlinks", expectArgs{"/var/lib/planterette/base/debian:f92c9052/root"}, "/var/lib/planterette/base/debian:f92c9052/root", nil},
			{"evalSymlinks", expectArgs{"/var/lib/planterette/base/debian:f92c9052/run"}, "/var/lib/planterette/base/debian:f92c9052/run", nil},
			{"evalSymlinks", expectArgs{"/var/lib/planterette/base/debian:f92c9052/srv"}, "/var/lib/planterette/base/debian:f92c9052/srv", nil},
			{"evalSymlinks", expectArgs{"/var/lib/planterette/base/debian:f92c9052/sys"}, "/var/lib/planterette/base/debian:f92c9052/sys", nil},
			{"evalSymlinks", expectArgs{"/var/lib/planterette/base/debian:f92c9052/usr"}, "/var/lib/planterette/base/debian:f92c9052/usr", nil},
			{"evalSymlinks", expectArgs{"/var/lib/planterette/base/debian:f92c9052/var"}, "/var/lib/planterette/base/debian:f92c9052/var", nil},
		}, nil, []kexpect{
			{"verbosef", expectArgs{"%s %s", []any{"mounting", &BindMountOp{MustAbs("/var/lib/planterette/base/debian:f92c9052/usr/bin"), MustAbs("/var/lib/planterette/base/debian:f92c9052/bin"), MustAbs("/bin"), 0}}}, nil, nil}, {"stat", expectArgs{"/host/var/lib/planterette/base/debian:f92c9052/usr/bin"}, isDirFi(true), nil}, {"mkdirAll", expectArgs{"/sysroot/bin", os.FileMode(0700)}, nil, nil}, {"bindMount", expectArgs{"/host/var/lib/planterette/base/debian:f92c9052/usr/bin", "/sysroot/bin", uintptr(0x4005), false}, nil, nil},
			{"verbosef", expectArgs{"%s %s", []any{"mounting", &BindMountOp{MustAbs("/var/lib/planterette/base/debian:f92c9052/home"), MustAbs("/var/lib/planterette/base/debian:f92c9052/home"), MustAbs("/home"), 0}}}, nil, nil}, {"stat", expectArgs{"/host/var/lib/planterette/base/debian:f92c9052/home"}, isDirFi(true), nil}, {"mkdirAll", expectArgs{"/sysroot/home", os.FileMode(0700)}, nil, nil}, {"bindMount", expectArgs{"/host/var/lib/planterette/base/debian:f92c9052/home", "/sysroot/home", uintptr(0x4005), false}, nil, nil},
			{"verbosef", expectArgs{"%s %s", []any{"mounting", &BindMountOp{MustAbs("/var/lib/planterette/base/debian:f92c9052/lib64"), MustAbs("/var/lib/planterette/base/debian:f92c9052/lib64"), MustAbs("/lib64"), 0}}}, nil, nil}, {"stat", expectArgs{"/host/var/lib/planterette/base/debian:f92c9052/lib64"}, isDirFi(true), nil}, {"mkdirAll", expectArgs{"/sysroot/lib64", os.FileMode(0700)}, nil, nil}, {"bindMount", expectArgs{"/host/var/lib/planterette/base/debian:f92c9052/lib64", "/sysroot/lib64", uintptr(0x4005), false}, nil, nil},
			{"verbosef", expectArgs{"%s %s", []any{"mounting", &BindMountOp{MustAbs("/var/lib/planterette/base/debian:f92c9052/lost+found"), MustAbs("/var/lib/planterette/base/debian:f92c9052/lost+found"), MustAbs("/lost+found"), 0}}}, nil, nil}, {"stat", expectArgs{"/host/var/lib/planterette/base/debian:f92c9052/lost+found"}, isDirFi(true), nil}, {"mkdirAll", expectArgs{"/sysroot/lost+found", os.FileMode(0700)}, nil, nil}, {"bindMount", expectArgs{"/host/var/lib/planterette/base/debian:f92c9052/lost+found", "/sysroot/lost+found", uintptr(0x4005), false}, nil, nil},
			{"verbosef", expectArgs{"%s %s", []any{"mounting", &BindMountOp{MustAbs("/var/lib/planterette/base/debian:f92c9052/nix"), MustAbs("/var/lib/planterette/base/debian:f92c9052/nix"), MustAbs("/nix"), 0}}}, nil, nil}, {"stat", expectArgs{"/host/var/lib/planterette/base/debian:f92c9052/nix"}, isDirFi(true), nil}, {"mkdirAll", expectArgs{"/sysroot/nix", os.FileMode(0700)}, nil, nil}, {"bindMount", expectArgs{"/host/var/lib/planterette/base/debian:f92c9052/nix", "/sysroot/nix", uintptr(0x4005), false}, nil, nil},
			{"verbosef", expectArgs{"%s %s", []any{"mounting", &BindMountOp{MustAbs("/var/lib/planterette/base/debian:f92c9052/root"), MustAbs("/var/lib/planterette/base/debian:f92c9052/root"), MustAbs("/root"), 0}}}, nil, nil}, {"stat", expectArgs{"/host/var/lib/planterette/base/debian:f92c9052/root"}, isDirFi(true), nil}, {"mkdirAll", expectArgs{"/sysroot/root", os.FileMode(0700)}, nil, nil}, {"bindMount", expectArgs{"/host/var/lib/planterette/base/debian:f92c9052/root", "/sysroot/root", uintptr(0x4005), false}, nil, nil},
			{"verbosef", expectArgs{"%s %s", []any{"mounting", &BindMountOp{MustAbs("/var/lib/planterette/base/debian:f92c9052/run"), MustAbs("/var/lib/planterette/base/debian:f92c9052/run"), MustAbs("/run"), 0}}}, nil, nil}, {"stat", expectArgs{"/host/var/lib/planterette/base/debian:f92c9052/run"}, isDirFi(true), nil}, {"mkdirAll", expectArgs{"/sysroot/run", os.FileMode(0700)}, nil, nil}, {"bindMount", expectArgs{"/host/var/lib/planterette/base/debian:f92c9052/run", "/sysroot/run", uintptr(0x4005), false}, nil, nil},
			{"verbosef", expectArgs{"%s %s", []any{"mounting", &BindMountOp{MustAbs("/var/lib/planterette/base/debian:f92c9052/srv"), MustAbs("/var/lib/planterette/base/debian:f92c9052/srv"), MustAbs("/srv"), 0}}}, nil, nil}, {"stat", expectArgs{"/host/var/lib/planterette/base/debian:f92c9052/srv"}, isDirFi(true), nil}, {"mkdirAll", expectArgs{"/sysroot/srv", os.FileMode(0700)}, nil, nil}, {"bindMount", expectArgs{"/host/var/lib/planterette/base/debian:f92c9052/srv", "/sysroot/srv", uintptr(0x4005), false}, nil, nil},
			{"verbosef", expectArgs{"%s %s", []any{"mounting", &BindMountOp{MustAbs("/var/lib/planterette/base/debian:f92c9052/sys"), MustAbs("/var/lib/planterette/base/debian:f92c9052/sys"), MustAbs("/sys"), 0}}}, nil, nil}, {"stat", expectArgs{"/host/var/lib/planterette/base/debian:f92c9052/sys"}, isDirFi(true), nil}, {"mkdirAll", expectArgs{"/sysroot/sys", os.FileMode(0700)}, nil, nil}, {"bindMount", expectArgs{"/host/var/lib/planterette/base/debian:f92c9052/sys", "/sysroot/sys", uintptr(0x4005), false}, nil, nil},
			{"verbosef", expectArgs{"%s %s", []any{"mounting", &BindMountOp{MustAbs("/var/lib/planterette/base/debian:f92c9052/usr"), MustAbs("/var/lib/planterette/base/debian:f92c9052/usr"), MustAbs("/usr"), 0}}}, nil, nil}, {"stat", expectArgs{"/host/var/lib/planterette/base/debian:f92c9052/usr"}, isDirFi(true), nil}, {"mkdirAll", expectArgs{"/sysroot/usr", os.FileMode(0700)}, nil, nil}, {"bindMount", expectArgs{"/host/var/lib/planterette/base/debian:f92c9052/usr", "/sysroot/usr", uintptr(0x4005), false}, nil, nil},
			{"verbosef", expectArgs{"%s %s", []any{"mounting", &BindMountOp{MustAbs("/var/lib/planterette/base/debian:f92c9052/var"), MustAbs("/var/lib/planterette/base/debian:f92c9052/var"), MustAbs("/var"), 0}}}, nil, nil}, {"stat", expectArgs{"/host/var/lib/planterette/base/debian:f92c9052/var"}, isDirFi(true), nil}, {"mkdirAll", expectArgs{"/sysroot/var", os.FileMode(0700)}, nil, nil}, {"bindMount", expectArgs{"/host/var/lib/planterette/base/debian:f92c9052/var", "/sysroot/var", uintptr(0x4005), false}, nil, nil},
		}, nil},
	})

	checkOpsValid(t, []opValidTestCase{
		{"nil", (*AutoRootOp)(nil), false},
		{"zero", new(AutoRootOp), false},
		{"valid", &AutoRootOp{Host: MustAbs("/")}, true},
	})

	checkOpsBuilder(t, []opsBuilderTestCase{
		{"pd", new(Ops).Root(MustAbs("/"), "048090b6ed8f9ebb10e275ff5d8c0659", BindWritable), Ops{
			&AutoRootOp{
				Host:   MustAbs("/"),
				Prefix: "048090b6ed8f9ebb10e275ff5d8c0659",
				Flags:  BindWritable,
			},
		}},
	})

	checkOpIs(t, []opIsTestCase{
		{"zero", new(AutoRootOp), new(AutoRootOp), false},

		{"internal ne", &AutoRootOp{
			Host:   MustAbs("/"),
			Prefix: ":3",
			Flags:  BindWritable,
		}, &AutoRootOp{
			Host:     MustAbs("/"),
			Prefix:   ":3",
			Flags:    BindWritable,
			resolved: []Op{new(BindMountOp)},
		}, true},

		{"prefix differs", &AutoRootOp{
			Host:   MustAbs("/"),
			Prefix: "\x00",
			Flags:  BindWritable,
		}, &AutoRootOp{
			Host:   MustAbs("/"),
			Prefix: ":3",
			Flags:  BindWritable,
		}, false},

		{"flags differs", &AutoRootOp{
			Host:   MustAbs("/"),
			Prefix: ":3",
			Flags:  BindWritable | BindDevice,
		}, &AutoRootOp{
			Host:   MustAbs("/"),
			Prefix: ":3",
			Flags:  BindWritable,
		}, false},

		{"host differs", &AutoRootOp{
			Host:   MustAbs("/tmp/"),
			Prefix: ":3",
			Flags:  BindWritable,
		}, &AutoRootOp{
			Host:   MustAbs("/"),
			Prefix: ":3",
			Flags:  BindWritable,
		}, false},

		{"equals", &AutoRootOp{
			Host:   MustAbs("/"),
			Prefix: ":3",
			Flags:  BindWritable,
		}, &AutoRootOp{
			Host:   MustAbs("/"),
			Prefix: ":3",
			Flags:  BindWritable,
		}, true},
	})

	checkOpMeta(t, []opMetaTestCase{
		{"root", &AutoRootOp{
			Host:   MustAbs("/"),
			Prefix: ":3",
			Flags:  BindWritable,
		}, "setting up", `auto root "/" prefix :3 flags 0x2`},
	})
}

func TestIsAutoRootBindable(t *testing.T) {
	testCases := []struct {
		name string
		want bool
	}{
		{"proc", false},
		{"dev", false},
		{"tmp", false},
		{"mnt", false},
		{"etc", false},
		{"", false},

		{"var", true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsAutoRootBindable(tc.name); got != tc.want {
				t.Errorf("IsAutoRootBindable: %v, want %v", got, tc.want)
			}
		})
	}
}
