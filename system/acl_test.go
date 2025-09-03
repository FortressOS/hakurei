package system

import (
	"os"
	"syscall"
	"testing"

	"hakurei.app/container/stub"
	"hakurei.app/system/acl"
)

func TestACLUpdateOp(t *testing.T) {
	checkOpBehaviour(t, []opBehaviourTestCase{
		{"apply aclUpdate", 0xdeadbeef, 0xff,
			&ACLUpdateOp{Process, "/proc/nonexistent", []acl.Perm{acl.Read, acl.Write, acl.Execute}}, []stub.Call{
				call("verbose", stub.ExpectArgs{[]any{"applying ACL", &ACLUpdateOp{Process, "/proc/nonexistent", []acl.Perm{acl.Read, acl.Write, acl.Execute}}}}, nil, nil),
				call("aclUpdate", stub.ExpectArgs{"/proc/nonexistent", 0xdeadbeef, []acl.Perm{acl.Read, acl.Write, acl.Execute}}, nil, stub.UniqueError(1)),
			}, &OpError{Op: "acl", Err: stub.UniqueError(1)}, nil, nil},

		{"revert aclUpdate", 0xdeadbeef, 0xff,
			&ACLUpdateOp{Process, "/proc/nonexistent", []acl.Perm{acl.Read, acl.Write, acl.Execute}}, []stub.Call{
				call("verbose", stub.ExpectArgs{[]any{"applying ACL", &ACLUpdateOp{Process, "/proc/nonexistent", []acl.Perm{acl.Read, acl.Write, acl.Execute}}}}, nil, nil),
				call("aclUpdate", stub.ExpectArgs{"/proc/nonexistent", 0xdeadbeef, []acl.Perm{acl.Read, acl.Write, acl.Execute}}, nil, nil),
			}, nil, []stub.Call{
				call("verbose", stub.ExpectArgs{[]any{"stripping ACL", &ACLUpdateOp{Process, "/proc/nonexistent", []acl.Perm{acl.Read, acl.Write, acl.Execute}}}}, nil, nil),
				call("aclUpdate", stub.ExpectArgs{"/proc/nonexistent", 0xdeadbeef, ([]acl.Perm)(nil)}, nil, stub.UniqueError(0)),
			}, &OpError{Op: "acl", Err: stub.UniqueError(0), Revert: true}},

		{"success revert skip", 0xdeadbeef, Process,
			&ACLUpdateOp{User, "/proc/nonexistent", []acl.Perm{acl.Read, acl.Write, acl.Execute}}, []stub.Call{
				call("verbose", stub.ExpectArgs{[]any{"applying ACL", &ACLUpdateOp{User, "/proc/nonexistent", []acl.Perm{acl.Read, acl.Write, acl.Execute}}}}, nil, nil),
				call("aclUpdate", stub.ExpectArgs{"/proc/nonexistent", 0xdeadbeef, []acl.Perm{acl.Read, acl.Write, acl.Execute}}, nil, nil),
			}, nil, []stub.Call{
				call("verbose", stub.ExpectArgs{[]any{"skipping ACL", &ACLUpdateOp{User, "/proc/nonexistent", []acl.Perm{acl.Read, acl.Write, acl.Execute}}}}, nil, nil),
			}, nil},

		{"success revert aclUpdate ENOENT", 0xdeadbeef, 0xff,
			&ACLUpdateOp{Process, "/proc/nonexistent", []acl.Perm{acl.Read, acl.Write, acl.Execute}}, []stub.Call{
				call("verbose", stub.ExpectArgs{[]any{"applying ACL", &ACLUpdateOp{Process, "/proc/nonexistent", []acl.Perm{acl.Read, acl.Write, acl.Execute}}}}, nil, nil),
				call("aclUpdate", stub.ExpectArgs{"/proc/nonexistent", 0xdeadbeef, []acl.Perm{acl.Read, acl.Write, acl.Execute}}, nil, nil),
			}, nil, []stub.Call{
				call("verbose", stub.ExpectArgs{[]any{"stripping ACL", &ACLUpdateOp{Process, "/proc/nonexistent", []acl.Perm{acl.Read, acl.Write, acl.Execute}}}}, nil, nil),
				call("aclUpdate", stub.ExpectArgs{"/proc/nonexistent", 0xdeadbeef, ([]acl.Perm)(nil)}, nil, &os.PathError{Op: "acl_get_file", Path: "/proc/nonexistent", Err: syscall.ENOENT}),
				call("verbosef", stub.ExpectArgs{"target of ACL %s no longer exists", []any{&ACLUpdateOp{Process, "/proc/nonexistent", []acl.Perm{acl.Read, acl.Write, acl.Execute}}}}, nil, nil),
			}, nil},

		{"success", 0xdeadbeef, 0xff,
			&ACLUpdateOp{Process, "/proc/nonexistent", []acl.Perm{acl.Read, acl.Write, acl.Execute}}, []stub.Call{
				call("verbose", stub.ExpectArgs{[]any{"applying ACL", &ACLUpdateOp{Process, "/proc/nonexistent", []acl.Perm{acl.Read, acl.Write, acl.Execute}}}}, nil, nil),
				call("aclUpdate", stub.ExpectArgs{"/proc/nonexistent", 0xdeadbeef, []acl.Perm{acl.Read, acl.Write, acl.Execute}}, nil, nil),
			}, nil, []stub.Call{
				call("verbose", stub.ExpectArgs{[]any{"stripping ACL", &ACLUpdateOp{Process, "/proc/nonexistent", []acl.Perm{acl.Read, acl.Write, acl.Execute}}}}, nil, nil),
				call("aclUpdate", stub.ExpectArgs{"/proc/nonexistent", 0xdeadbeef, ([]acl.Perm)(nil)}, nil, nil),
			}, nil},
	})

	checkOpsBuilder(t, "UpdatePerm", []opsBuilderTestCase{
		{"simple",
			0xdeadbeef,
			func(sys *I) {
				sys.
					UpdatePerm("/run/user/1971/hakurei", acl.Execute).
					UpdatePerm("/tmp/hakurei.0/tmpdir/150", acl.Read, acl.Write, acl.Execute)
			}, []Op{
				&ACLUpdateOp{Process, "/run/user/1971/hakurei", []acl.Perm{acl.Execute}},
				&ACLUpdateOp{Process, "/tmp/hakurei.0/tmpdir/150", []acl.Perm{acl.Read, acl.Write, acl.Execute}},
			}, stub.Expect{}},
	})
	checkOpsBuilder(t, "UpdatePermType", []opsBuilderTestCase{
		{"tmpdirp", 0xdeadbeef, func(sys *I) {
			sys.UpdatePermType(User, "/tmp/hakurei.0/tmpdir", acl.Execute)
		}, []Op{
			&ACLUpdateOp{User, "/tmp/hakurei.0/tmpdir", []acl.Perm{acl.Execute}},
		}, stub.Expect{}},

		{"tmpdir", 0xdeadbeef, func(sys *I) {
			sys.UpdatePermType(User, "/tmp/hakurei.0/tmpdir/150", acl.Read, acl.Write, acl.Execute)
		}, []Op{
			&ACLUpdateOp{User, "/tmp/hakurei.0/tmpdir/150", []acl.Perm{acl.Read, acl.Write, acl.Execute}},
		}, stub.Expect{}},

		{"share", 0xdeadbeef, func(sys *I) {
			sys.UpdatePermType(Process, "/run/user/1971/hakurei/fcb8a12f7c482d183ade8288c3de78b5", acl.Execute)
		}, []Op{
			&ACLUpdateOp{Process, "/run/user/1971/hakurei/fcb8a12f7c482d183ade8288c3de78b5", []acl.Perm{acl.Execute}},
		}, stub.Expect{}},

		{"passwd", 0xdeadbeef, func(sys *I) {
			sys.
				UpdatePermType(Process, "/tmp/hakurei.0/fcb8a12f7c482d183ade8288c3de78b5/passwd", acl.Read).
				UpdatePermType(Process, "/tmp/hakurei.0/fcb8a12f7c482d183ade8288c3de78b5/group", acl.Read)
		}, []Op{
			&ACLUpdateOp{Process, "/tmp/hakurei.0/fcb8a12f7c482d183ade8288c3de78b5/passwd", []acl.Perm{acl.Read}},
			&ACLUpdateOp{Process, "/tmp/hakurei.0/fcb8a12f7c482d183ade8288c3de78b5/group", []acl.Perm{acl.Read}},
		}, stub.Expect{}},

		{"wayland", 0xdeadbeef, func(sys *I) {
			sys.UpdatePermType(EWayland, "/run/user/1971/wayland-0", acl.Read, acl.Write, acl.Execute)
		}, []Op{
			&ACLUpdateOp{EWayland, "/run/user/1971/wayland-0", []acl.Perm{acl.Read, acl.Write, acl.Execute}},
		}, stub.Expect{}},
	})

	checkOpIs(t, []opIsTestCase{
		{"nil", (*ACLUpdateOp)(nil), (*ACLUpdateOp)(nil), false},
		{"zero", new(ACLUpdateOp), new(ACLUpdateOp), true},

		{"et differs",
			&ACLUpdateOp{
				EWayland, "/run/user/1971/wayland-0",
				[]acl.Perm{acl.Read, acl.Write, acl.Execute},
			}, &ACLUpdateOp{
				EX11, "/run/user/1971/wayland-0",
				[]acl.Perm{acl.Read, acl.Write, acl.Execute},
			}, false},

		{"path differs", &ACLUpdateOp{
			EWayland, "/run/user/1971/wayland-0",
			[]acl.Perm{acl.Read, acl.Write, acl.Execute},
		}, &ACLUpdateOp{
			EWayland, "/run/user/1971/wayland-1",
			[]acl.Perm{acl.Read, acl.Write, acl.Execute},
		}, false},

		{"perms differs", &ACLUpdateOp{
			EWayland, "/run/user/1971/wayland-0",
			[]acl.Perm{acl.Read, acl.Write, acl.Execute},
		}, &ACLUpdateOp{
			EWayland, "/run/user/1971/wayland-0",
			[]acl.Perm{acl.Read, acl.Write},
		}, false},

		{"equals", &ACLUpdateOp{
			EWayland, "/run/user/1971/wayland-0",
			[]acl.Perm{acl.Read, acl.Write, acl.Execute},
		}, &ACLUpdateOp{
			EWayland, "/run/user/1971/wayland-0",
			[]acl.Perm{acl.Read, acl.Write, acl.Execute},
		}, true},
	})

	checkOpMeta(t, []opMetaTestCase{
		{"clear",
			&ACLUpdateOp{Process, "/proc/nonexistent", []acl.Perm{}},
			Process, "/proc/nonexistent",
			`--- type: process path: "/proc/nonexistent"`},

		{"read",
			&ACLUpdateOp{User, "/tmp/hakurei.0/27d81d567f8fae7f33278eec45da9446/0", []acl.Perm{acl.Read}},
			User, "/tmp/hakurei.0/27d81d567f8fae7f33278eec45da9446/0",
			`r-- type: user path: "/tmp/hakurei.0/27d81d567f8fae7f33278eec45da9446/0"`},

		{"write",
			&ACLUpdateOp{User, "/tmp/hakurei.0/27d81d567f8fae7f33278eec45da9446/1", []acl.Perm{acl.Write}},
			User, "/tmp/hakurei.0/27d81d567f8fae7f33278eec45da9446/1",
			`-w- type: user path: "/tmp/hakurei.0/27d81d567f8fae7f33278eec45da9446/1"`},

		{"execute",
			&ACLUpdateOp{User, "/tmp/hakurei.0/27d81d567f8fae7f33278eec45da9446/2", []acl.Perm{acl.Execute}},
			User, "/tmp/hakurei.0/27d81d567f8fae7f33278eec45da9446/2",
			`--x type: user path: "/tmp/hakurei.0/27d81d567f8fae7f33278eec45da9446/2"`},

		{"wayland",
			&ACLUpdateOp{EWayland, "/tmp/hakurei.0/27d81d567f8fae7f33278eec45da9446/wayland", []acl.Perm{acl.Read, acl.Write}},
			EWayland, "/tmp/hakurei.0/27d81d567f8fae7f33278eec45da9446/wayland",
			`rw- type: wayland path: "/tmp/hakurei.0/27d81d567f8fae7f33278eec45da9446/wayland"`},

		{"x11",
			&ACLUpdateOp{EX11, "/tmp/.X11-unix/X0", []acl.Perm{acl.Read, acl.Execute}},
			EX11, "/tmp/.X11-unix/X0",
			`r-x type: x11 path: "/tmp/.X11-unix/X0"`},

		{"dbus",
			&ACLUpdateOp{EDBus, "/tmp/hakurei.0/27d81d567f8fae7f33278eec45da9446/bus", []acl.Perm{acl.Write, acl.Execute}},
			EDBus, "/tmp/hakurei.0/27d81d567f8fae7f33278eec45da9446/bus",
			`-wx type: dbus path: "/tmp/hakurei.0/27d81d567f8fae7f33278eec45da9446/bus"`},

		{"pulseaudio",
			&ACLUpdateOp{EPulse, "/run/user/1971/hakurei/27d81d567f8fae7f33278eec45da9446/pulse", []acl.Perm{acl.Read, acl.Write, acl.Execute}},
			EPulse, "/run/user/1971/hakurei/27d81d567f8fae7f33278eec45da9446/pulse",
			`rwx type: pulseaudio path: "/run/user/1971/hakurei/27d81d567f8fae7f33278eec45da9446/pulse"`},
	})
}
