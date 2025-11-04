package container

import (
	"errors"
	"os"
	"testing"

	"hakurei.app/container/check"
	"hakurei.app/container/std"
	"hakurei.app/container/stub"
	"hakurei.app/message"
)

func TestAutoRootOp(t *testing.T) {
	t.Run("nonrepeatable", func(t *testing.T) {
		t.Parallel()
		wantErr := OpRepeatError("autoroot")
		if err := new(AutoRootOp).apply(&setupState{nonrepeatable: nrAutoRoot}, nil); !errors.Is(err, wantErr) {
			t.Errorf("apply: error = %v, want %v", err, wantErr)
		}
	})

	checkOpBehaviour(t, []opBehaviourTestCase{
		{"readdir", &Params{ParentPerm: 0750}, &AutoRootOp{
			Host:  check.MustAbs("/"),
			Flags: std.BindWritable,
		}, []stub.Call{
			call("readdir", stub.ExpectArgs{"/"}, stubDir(), stub.UniqueError(2)),
		}, stub.UniqueError(2), nil, nil},

		{"early", &Params{ParentPerm: 0750}, &AutoRootOp{
			Host:  check.MustAbs("/"),
			Flags: std.BindWritable,
		}, []stub.Call{
			call("readdir", stub.ExpectArgs{"/"}, stubDir("bin", "dev", "etc", "home", "lib64",
				"lost+found", "mnt", "nix", "proc", "root", "run", "srv", "sys", "tmp", "usr", "var"), nil),
			call("evalSymlinks", stub.ExpectArgs{"/bin"}, "", stub.UniqueError(1)),
		}, stub.UniqueError(1), nil, nil},

		{"apply", &Params{ParentPerm: 0750}, &AutoRootOp{
			Host:  check.MustAbs("/"),
			Flags: std.BindWritable,
		}, []stub.Call{
			call("readdir", stub.ExpectArgs{"/"}, stubDir("bin", "dev", "etc", "home", "lib64",
				"lost+found", "mnt", "nix", "proc", "root", "run", "srv", "sys", "tmp", "usr", "var"), nil),
			call("evalSymlinks", stub.ExpectArgs{"/bin"}, "/usr/bin", nil),
			call("evalSymlinks", stub.ExpectArgs{"/home"}, "/home", nil),
			call("evalSymlinks", stub.ExpectArgs{"/lib64"}, "/lib64", nil),
			call("evalSymlinks", stub.ExpectArgs{"/lost+found"}, "/lost+found", nil),
			call("evalSymlinks", stub.ExpectArgs{"/nix"}, "/nix", nil),
			call("evalSymlinks", stub.ExpectArgs{"/root"}, "/root", nil),
			call("evalSymlinks", stub.ExpectArgs{"/run"}, "/run", nil),
			call("evalSymlinks", stub.ExpectArgs{"/srv"}, "/srv", nil),
			call("evalSymlinks", stub.ExpectArgs{"/sys"}, "/sys", nil),
			call("evalSymlinks", stub.ExpectArgs{"/usr"}, "/usr", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var"}, "/var", nil),
		}, nil, []stub.Call{
			call("stat", stub.ExpectArgs{"/host/usr/bin"}, isDirFi(false), stub.UniqueError(0)),
		}, stub.UniqueError(0)},

		{"success pd", &Params{ParentPerm: 0750}, &AutoRootOp{
			Host:  check.MustAbs("/"),
			Flags: std.BindWritable,
		}, []stub.Call{
			call("readdir", stub.ExpectArgs{"/"}, stubDir("bin", "dev", "etc", "home", "lib64",
				"lost+found", "mnt", "nix", "proc", "root", "run", "srv", "sys", "tmp", "usr", "var"), nil),
			call("evalSymlinks", stub.ExpectArgs{"/bin"}, "/usr/bin", nil),
			call("evalSymlinks", stub.ExpectArgs{"/home"}, "/home", nil),
			call("evalSymlinks", stub.ExpectArgs{"/lib64"}, "/lib64", nil),
			call("evalSymlinks", stub.ExpectArgs{"/lost+found"}, "/lost+found", nil),
			call("evalSymlinks", stub.ExpectArgs{"/nix"}, "/nix", nil),
			call("evalSymlinks", stub.ExpectArgs{"/root"}, "/root", nil),
			call("evalSymlinks", stub.ExpectArgs{"/run"}, "/run", nil),
			call("evalSymlinks", stub.ExpectArgs{"/srv"}, "/srv", nil),
			call("evalSymlinks", stub.ExpectArgs{"/sys"}, "/sys", nil),
			call("evalSymlinks", stub.ExpectArgs{"/usr"}, "/usr", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var"}, "/var", nil),
		}, nil, []stub.Call{
			call("stat", stub.ExpectArgs{"/host/usr/bin"}, isDirFi(true), nil), call("mkdirAll", stub.ExpectArgs{"/sysroot/bin", os.FileMode(0700)}, nil, nil), call("verbosef", stub.ExpectArgs{"mounting %q on %q flags %#x", []any{"/host/usr/bin", "/sysroot/bin", uintptr(0x4004)}}, nil, nil), call("bindMount", stub.ExpectArgs{"/host/usr/bin", "/sysroot/bin", uintptr(0x4004), false}, nil, nil),
			call("stat", stub.ExpectArgs{"/host/home"}, isDirFi(true), nil), call("mkdirAll", stub.ExpectArgs{"/sysroot/home", os.FileMode(0700)}, nil, nil), call("verbosef", stub.ExpectArgs{"mounting %q flags %#x", []any{"/sysroot/home", uintptr(0x4004)}}, nil, nil), call("bindMount", stub.ExpectArgs{"/host/home", "/sysroot/home", uintptr(0x4004), false}, nil, nil),
			call("stat", stub.ExpectArgs{"/host/lib64"}, isDirFi(true), nil), call("mkdirAll", stub.ExpectArgs{"/sysroot/lib64", os.FileMode(0700)}, nil, nil), call("verbosef", stub.ExpectArgs{"mounting %q flags %#x", []any{"/sysroot/lib64", uintptr(0x4004)}}, nil, nil), call("bindMount", stub.ExpectArgs{"/host/lib64", "/sysroot/lib64", uintptr(0x4004), false}, nil, nil),
			call("stat", stub.ExpectArgs{"/host/lost+found"}, isDirFi(true), nil), call("mkdirAll", stub.ExpectArgs{"/sysroot/lost+found", os.FileMode(0700)}, nil, nil), call("verbosef", stub.ExpectArgs{"mounting %q flags %#x", []any{"/sysroot/lost+found", uintptr(0x4004)}}, nil, nil), call("bindMount", stub.ExpectArgs{"/host/lost+found", "/sysroot/lost+found", uintptr(0x4004), false}, nil, nil),
			call("stat", stub.ExpectArgs{"/host/nix"}, isDirFi(true), nil), call("mkdirAll", stub.ExpectArgs{"/sysroot/nix", os.FileMode(0700)}, nil, nil), call("verbosef", stub.ExpectArgs{"mounting %q flags %#x", []any{"/sysroot/nix", uintptr(0x4004)}}, nil, nil), call("bindMount", stub.ExpectArgs{"/host/nix", "/sysroot/nix", uintptr(0x4004), false}, nil, nil),
			call("stat", stub.ExpectArgs{"/host/root"}, isDirFi(true), nil), call("mkdirAll", stub.ExpectArgs{"/sysroot/root", os.FileMode(0700)}, nil, nil), call("verbosef", stub.ExpectArgs{"mounting %q flags %#x", []any{"/sysroot/root", uintptr(0x4004)}}, nil, nil), call("bindMount", stub.ExpectArgs{"/host/root", "/sysroot/root", uintptr(0x4004), false}, nil, nil),
			call("stat", stub.ExpectArgs{"/host/run"}, isDirFi(true), nil), call("mkdirAll", stub.ExpectArgs{"/sysroot/run", os.FileMode(0700)}, nil, nil), call("verbosef", stub.ExpectArgs{"mounting %q flags %#x", []any{"/sysroot/run", uintptr(0x4004)}}, nil, nil), call("bindMount", stub.ExpectArgs{"/host/run", "/sysroot/run", uintptr(0x4004), false}, nil, nil),
			call("stat", stub.ExpectArgs{"/host/srv"}, isDirFi(true), nil), call("mkdirAll", stub.ExpectArgs{"/sysroot/srv", os.FileMode(0700)}, nil, nil), call("verbosef", stub.ExpectArgs{"mounting %q flags %#x", []any{"/sysroot/srv", uintptr(0x4004)}}, nil, nil), call("bindMount", stub.ExpectArgs{"/host/srv", "/sysroot/srv", uintptr(0x4004), false}, nil, nil),
			call("stat", stub.ExpectArgs{"/host/sys"}, isDirFi(true), nil), call("mkdirAll", stub.ExpectArgs{"/sysroot/sys", os.FileMode(0700)}, nil, nil), call("verbosef", stub.ExpectArgs{"mounting %q flags %#x", []any{"/sysroot/sys", uintptr(0x4004)}}, nil, nil), call("bindMount", stub.ExpectArgs{"/host/sys", "/sysroot/sys", uintptr(0x4004), false}, nil, nil),
			call("stat", stub.ExpectArgs{"/host/usr"}, isDirFi(true), nil), call("mkdirAll", stub.ExpectArgs{"/sysroot/usr", os.FileMode(0700)}, nil, nil), call("verbosef", stub.ExpectArgs{"mounting %q flags %#x", []any{"/sysroot/usr", uintptr(0x4004)}}, nil, nil), call("bindMount", stub.ExpectArgs{"/host/usr", "/sysroot/usr", uintptr(0x4004), false}, nil, nil),
			call("stat", stub.ExpectArgs{"/host/var"}, isDirFi(true), nil), call("mkdirAll", stub.ExpectArgs{"/sysroot/var", os.FileMode(0700)}, nil, nil), call("verbosef", stub.ExpectArgs{"mounting %q flags %#x", []any{"/sysroot/var", uintptr(0x4004)}}, nil, nil), call("bindMount", stub.ExpectArgs{"/host/var", "/sysroot/var", uintptr(0x4004), false}, nil, nil),
		}, nil},

		{"success", &Params{ParentPerm: 0750}, &AutoRootOp{
			Host: check.MustAbs("/var/lib/planterette/base/debian:f92c9052"),
		}, []stub.Call{
			call("readdir", stub.ExpectArgs{"/var/lib/planterette/base/debian:f92c9052"}, stubDir("bin", "dev", "etc", "home", "lib64",
				"lost+found", "mnt", "nix", "proc", "root", "run", "srv", "sys", "tmp", "usr", "var"), nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/planterette/base/debian:f92c9052/bin"}, "/var/lib/planterette/base/debian:f92c9052/usr/bin", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/planterette/base/debian:f92c9052/home"}, "/var/lib/planterette/base/debian:f92c9052/home", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/planterette/base/debian:f92c9052/lib64"}, "/var/lib/planterette/base/debian:f92c9052/lib64", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/planterette/base/debian:f92c9052/lost+found"}, "/var/lib/planterette/base/debian:f92c9052/lost+found", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/planterette/base/debian:f92c9052/nix"}, "/var/lib/planterette/base/debian:f92c9052/nix", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/planterette/base/debian:f92c9052/root"}, "/var/lib/planterette/base/debian:f92c9052/root", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/planterette/base/debian:f92c9052/run"}, "/var/lib/planterette/base/debian:f92c9052/run", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/planterette/base/debian:f92c9052/srv"}, "/var/lib/planterette/base/debian:f92c9052/srv", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/planterette/base/debian:f92c9052/sys"}, "/var/lib/planterette/base/debian:f92c9052/sys", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/planterette/base/debian:f92c9052/usr"}, "/var/lib/planterette/base/debian:f92c9052/usr", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/planterette/base/debian:f92c9052/var"}, "/var/lib/planterette/base/debian:f92c9052/var", nil),
		}, nil, []stub.Call{
			call("stat", stub.ExpectArgs{"/host/var/lib/planterette/base/debian:f92c9052/usr/bin"}, isDirFi(true), nil), call("mkdirAll", stub.ExpectArgs{"/sysroot/bin", os.FileMode(0700)}, nil, nil), call("verbosef", stub.ExpectArgs{"mounting %q on %q flags %#x", []any{"/host/var/lib/planterette/base/debian:f92c9052/usr/bin", "/sysroot/bin", uintptr(0x4005)}}, nil, nil), call("bindMount", stub.ExpectArgs{"/host/var/lib/planterette/base/debian:f92c9052/usr/bin", "/sysroot/bin", uintptr(0x4005), false}, nil, nil),
			call("stat", stub.ExpectArgs{"/host/var/lib/planterette/base/debian:f92c9052/home"}, isDirFi(true), nil), call("mkdirAll", stub.ExpectArgs{"/sysroot/home", os.FileMode(0700)}, nil, nil), call("verbosef", stub.ExpectArgs{"mounting %q on %q flags %#x", []any{"/host/var/lib/planterette/base/debian:f92c9052/home", "/sysroot/home", uintptr(0x4005)}}, nil, nil), call("bindMount", stub.ExpectArgs{"/host/var/lib/planterette/base/debian:f92c9052/home", "/sysroot/home", uintptr(0x4005), false}, nil, nil),
			call("stat", stub.ExpectArgs{"/host/var/lib/planterette/base/debian:f92c9052/lib64"}, isDirFi(true), nil), call("mkdirAll", stub.ExpectArgs{"/sysroot/lib64", os.FileMode(0700)}, nil, nil), call("verbosef", stub.ExpectArgs{"mounting %q on %q flags %#x", []any{"/host/var/lib/planterette/base/debian:f92c9052/lib64", "/sysroot/lib64", uintptr(0x4005)}}, nil, nil), call("bindMount", stub.ExpectArgs{"/host/var/lib/planterette/base/debian:f92c9052/lib64", "/sysroot/lib64", uintptr(0x4005), false}, nil, nil),
			call("stat", stub.ExpectArgs{"/host/var/lib/planterette/base/debian:f92c9052/lost+found"}, isDirFi(true), nil), call("mkdirAll", stub.ExpectArgs{"/sysroot/lost+found", os.FileMode(0700)}, nil, nil), call("verbosef", stub.ExpectArgs{"mounting %q on %q flags %#x", []any{"/host/var/lib/planterette/base/debian:f92c9052/lost+found", "/sysroot/lost+found", uintptr(0x4005)}}, nil, nil), call("bindMount", stub.ExpectArgs{"/host/var/lib/planterette/base/debian:f92c9052/lost+found", "/sysroot/lost+found", uintptr(0x4005), false}, nil, nil),
			call("stat", stub.ExpectArgs{"/host/var/lib/planterette/base/debian:f92c9052/nix"}, isDirFi(true), nil), call("mkdirAll", stub.ExpectArgs{"/sysroot/nix", os.FileMode(0700)}, nil, nil), call("verbosef", stub.ExpectArgs{"mounting %q on %q flags %#x", []any{"/host/var/lib/planterette/base/debian:f92c9052/nix", "/sysroot/nix", uintptr(0x4005)}}, nil, nil), call("bindMount", stub.ExpectArgs{"/host/var/lib/planterette/base/debian:f92c9052/nix", "/sysroot/nix", uintptr(0x4005), false}, nil, nil),
			call("stat", stub.ExpectArgs{"/host/var/lib/planterette/base/debian:f92c9052/root"}, isDirFi(true), nil), call("mkdirAll", stub.ExpectArgs{"/sysroot/root", os.FileMode(0700)}, nil, nil), call("verbosef", stub.ExpectArgs{"mounting %q on %q flags %#x", []any{"/host/var/lib/planterette/base/debian:f92c9052/root", "/sysroot/root", uintptr(0x4005)}}, nil, nil), call("bindMount", stub.ExpectArgs{"/host/var/lib/planterette/base/debian:f92c9052/root", "/sysroot/root", uintptr(0x4005), false}, nil, nil),
			call("stat", stub.ExpectArgs{"/host/var/lib/planterette/base/debian:f92c9052/run"}, isDirFi(true), nil), call("mkdirAll", stub.ExpectArgs{"/sysroot/run", os.FileMode(0700)}, nil, nil), call("verbosef", stub.ExpectArgs{"mounting %q on %q flags %#x", []any{"/host/var/lib/planterette/base/debian:f92c9052/run", "/sysroot/run", uintptr(0x4005)}}, nil, nil), call("bindMount", stub.ExpectArgs{"/host/var/lib/planterette/base/debian:f92c9052/run", "/sysroot/run", uintptr(0x4005), false}, nil, nil),
			call("stat", stub.ExpectArgs{"/host/var/lib/planterette/base/debian:f92c9052/srv"}, isDirFi(true), nil), call("mkdirAll", stub.ExpectArgs{"/sysroot/srv", os.FileMode(0700)}, nil, nil), call("verbosef", stub.ExpectArgs{"mounting %q on %q flags %#x", []any{"/host/var/lib/planterette/base/debian:f92c9052/srv", "/sysroot/srv", uintptr(0x4005)}}, nil, nil), call("bindMount", stub.ExpectArgs{"/host/var/lib/planterette/base/debian:f92c9052/srv", "/sysroot/srv", uintptr(0x4005), false}, nil, nil),
			call("stat", stub.ExpectArgs{"/host/var/lib/planterette/base/debian:f92c9052/sys"}, isDirFi(true), nil), call("mkdirAll", stub.ExpectArgs{"/sysroot/sys", os.FileMode(0700)}, nil, nil), call("verbosef", stub.ExpectArgs{"mounting %q on %q flags %#x", []any{"/host/var/lib/planterette/base/debian:f92c9052/sys", "/sysroot/sys", uintptr(0x4005)}}, nil, nil), call("bindMount", stub.ExpectArgs{"/host/var/lib/planterette/base/debian:f92c9052/sys", "/sysroot/sys", uintptr(0x4005), false}, nil, nil),
			call("stat", stub.ExpectArgs{"/host/var/lib/planterette/base/debian:f92c9052/usr"}, isDirFi(true), nil), call("mkdirAll", stub.ExpectArgs{"/sysroot/usr", os.FileMode(0700)}, nil, nil), call("verbosef", stub.ExpectArgs{"mounting %q on %q flags %#x", []any{"/host/var/lib/planterette/base/debian:f92c9052/usr", "/sysroot/usr", uintptr(0x4005)}}, nil, nil), call("bindMount", stub.ExpectArgs{"/host/var/lib/planterette/base/debian:f92c9052/usr", "/sysroot/usr", uintptr(0x4005), false}, nil, nil),
			call("stat", stub.ExpectArgs{"/host/var/lib/planterette/base/debian:f92c9052/var"}, isDirFi(true), nil), call("mkdirAll", stub.ExpectArgs{"/sysroot/var", os.FileMode(0700)}, nil, nil), call("verbosef", stub.ExpectArgs{"mounting %q on %q flags %#x", []any{"/host/var/lib/planterette/base/debian:f92c9052/var", "/sysroot/var", uintptr(0x4005)}}, nil, nil), call("bindMount", stub.ExpectArgs{"/host/var/lib/planterette/base/debian:f92c9052/var", "/sysroot/var", uintptr(0x4005), false}, nil, nil),
		}, nil},
	})

	checkOpsValid(t, []opValidTestCase{
		{"nil", (*AutoRootOp)(nil), false},
		{"zero", new(AutoRootOp), false},
		{"valid", &AutoRootOp{Host: check.MustAbs("/")}, true},
	})

	checkOpsBuilder(t, []opsBuilderTestCase{
		{"pd", new(Ops).Root(check.MustAbs("/"), std.BindWritable), Ops{
			&AutoRootOp{
				Host:  check.MustAbs("/"),
				Flags: std.BindWritable,
			},
		}},
	})

	checkOpIs(t, []opIsTestCase{
		{"zero", new(AutoRootOp), new(AutoRootOp), false},

		{"internal ne", &AutoRootOp{
			Host:  check.MustAbs("/"),
			Flags: std.BindWritable,
		}, &AutoRootOp{
			Host:     check.MustAbs("/"),
			Flags:    std.BindWritable,
			resolved: []*BindMountOp{new(BindMountOp)},
		}, true},

		{"flags differs", &AutoRootOp{
			Host:  check.MustAbs("/"),
			Flags: std.BindWritable | std.BindDevice,
		}, &AutoRootOp{
			Host:  check.MustAbs("/"),
			Flags: std.BindWritable,
		}, false},

		{"host differs", &AutoRootOp{
			Host:  check.MustAbs("/tmp/"),
			Flags: std.BindWritable,
		}, &AutoRootOp{
			Host:  check.MustAbs("/"),
			Flags: std.BindWritable,
		}, false},

		{"equals", &AutoRootOp{
			Host:  check.MustAbs("/"),
			Flags: std.BindWritable,
		}, &AutoRootOp{
			Host:  check.MustAbs("/"),
			Flags: std.BindWritable,
		}, true},
	})

	checkOpMeta(t, []opMetaTestCase{
		{"root", &AutoRootOp{
			Host:  check.MustAbs("/"),
			Flags: std.BindWritable,
		}, "setting up", `auto root "/" flags 0x2`},
	})
}

func TestIsAutoRootBindable(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		want bool
		log  bool
	}{
		{"proc", false, false},
		{"dev", false, false},
		{"tmp", false, false},
		{"mnt", false, false},
		{"etc", false, false},
		{"", false, true},

		{"var", true, false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var msg message.Msg
			if tc.log {
				msg = &kstub{nil, stub.New(t, func(s *stub.Stub[syscallDispatcher]) syscallDispatcher { panic("unreachable") }, stub.Expect{Calls: []stub.Call{
					call("verbose", stub.ExpectArgs{[]any{"got unexpected root entry"}}, nil, nil),
				}})}
			}
			if got := IsAutoRootBindable(msg, tc.name); got != tc.want {
				t.Errorf("IsAutoRootBindable: %v, want %v", got, tc.want)
			}
		})
	}
}
