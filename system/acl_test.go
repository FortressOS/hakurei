package system

import (
	"os"
	"syscall"
	"testing"

	"hakurei.app/container/stub"
	"hakurei.app/hst"
	"hakurei.app/system/acl"
)

func TestACLUpdateOp(t *testing.T) {
	checkOpBehaviour(t, []opBehaviourTestCase{
		{"apply aclUpdate", 0xdeadbeef, 0xff,
			&aclUpdateOp{Process, "/proc/nonexistent", []acl.Perm{acl.Read, acl.Write, acl.Execute}}, []stub.Call{
				call("verbose", stub.ExpectArgs{[]any{"applying ACL", &aclUpdateOp{Process, "/proc/nonexistent", []acl.Perm{acl.Read, acl.Write, acl.Execute}}}}, nil, nil),
				call("aclUpdate", stub.ExpectArgs{"/proc/nonexistent", 0xdeadbeef, []acl.Perm{acl.Read, acl.Write, acl.Execute}}, nil, stub.UniqueError(1)),
			}, &OpError{Op: "acl", Err: stub.UniqueError(1)}, nil, nil},

		{"revert aclUpdate", 0xdeadbeef, 0xff,
			&aclUpdateOp{Process, "/proc/nonexistent", []acl.Perm{acl.Read, acl.Write, acl.Execute}}, []stub.Call{
				call("verbose", stub.ExpectArgs{[]any{"applying ACL", &aclUpdateOp{Process, "/proc/nonexistent", []acl.Perm{acl.Read, acl.Write, acl.Execute}}}}, nil, nil),
				call("aclUpdate", stub.ExpectArgs{"/proc/nonexistent", 0xdeadbeef, []acl.Perm{acl.Read, acl.Write, acl.Execute}}, nil, nil),
			}, nil, []stub.Call{
				call("verbose", stub.ExpectArgs{[]any{"stripping ACL", &aclUpdateOp{Process, "/proc/nonexistent", []acl.Perm{acl.Read, acl.Write, acl.Execute}}}}, nil, nil),
				call("aclUpdate", stub.ExpectArgs{"/proc/nonexistent", 0xdeadbeef, ([]acl.Perm)(nil)}, nil, stub.UniqueError(0)),
			}, &OpError{Op: "acl", Err: stub.UniqueError(0), Revert: true}},

		{"success revert skip", 0xdeadbeef, Process,
			&aclUpdateOp{User, "/proc/nonexistent", []acl.Perm{acl.Read, acl.Write, acl.Execute}}, []stub.Call{
				call("verbose", stub.ExpectArgs{[]any{"applying ACL", &aclUpdateOp{User, "/proc/nonexistent", []acl.Perm{acl.Read, acl.Write, acl.Execute}}}}, nil, nil),
				call("aclUpdate", stub.ExpectArgs{"/proc/nonexistent", 0xdeadbeef, []acl.Perm{acl.Read, acl.Write, acl.Execute}}, nil, nil),
			}, nil, []stub.Call{
				call("verbose", stub.ExpectArgs{[]any{"skipping ACL", &aclUpdateOp{User, "/proc/nonexistent", []acl.Perm{acl.Read, acl.Write, acl.Execute}}}}, nil, nil),
			}, nil},

		{"success revert aclUpdate ENOENT", 0xdeadbeef, 0xff,
			&aclUpdateOp{Process, "/proc/nonexistent", []acl.Perm{acl.Read, acl.Write, acl.Execute}}, []stub.Call{
				call("verbose", stub.ExpectArgs{[]any{"applying ACL", &aclUpdateOp{Process, "/proc/nonexistent", []acl.Perm{acl.Read, acl.Write, acl.Execute}}}}, nil, nil),
				call("aclUpdate", stub.ExpectArgs{"/proc/nonexistent", 0xdeadbeef, []acl.Perm{acl.Read, acl.Write, acl.Execute}}, nil, nil),
			}, nil, []stub.Call{
				call("verbose", stub.ExpectArgs{[]any{"stripping ACL", &aclUpdateOp{Process, "/proc/nonexistent", []acl.Perm{acl.Read, acl.Write, acl.Execute}}}}, nil, nil),
				call("aclUpdate", stub.ExpectArgs{"/proc/nonexistent", 0xdeadbeef, ([]acl.Perm)(nil)}, nil, &os.PathError{Op: "acl_get_file", Path: "/proc/nonexistent", Err: syscall.ENOENT}),
				call("verbosef", stub.ExpectArgs{"target of ACL %s no longer exists", []any{&aclUpdateOp{Process, "/proc/nonexistent", []acl.Perm{acl.Read, acl.Write, acl.Execute}}}}, nil, nil),
			}, nil},

		{"success", 0xdeadbeef, 0xff,
			&aclUpdateOp{Process, "/proc/nonexistent", []acl.Perm{acl.Read, acl.Write, acl.Execute}}, []stub.Call{
				call("verbose", stub.ExpectArgs{[]any{"applying ACL", &aclUpdateOp{Process, "/proc/nonexistent", []acl.Perm{acl.Read, acl.Write, acl.Execute}}}}, nil, nil),
				call("aclUpdate", stub.ExpectArgs{"/proc/nonexistent", 0xdeadbeef, []acl.Perm{acl.Read, acl.Write, acl.Execute}}, nil, nil),
			}, nil, []stub.Call{
				call("verbose", stub.ExpectArgs{[]any{"stripping ACL", &aclUpdateOp{Process, "/proc/nonexistent", []acl.Perm{acl.Read, acl.Write, acl.Execute}}}}, nil, nil),
				call("aclUpdate", stub.ExpectArgs{"/proc/nonexistent", 0xdeadbeef, ([]acl.Perm)(nil)}, nil, nil),
			}, nil},
	})

	checkOpsBuilder(t, "UpdatePermType", []opsBuilderTestCase{
		{"simple",
			0xdeadbeef,
			func(_ *testing.T, sys *I) {
				sys.
					UpdatePerm("/run/user/1971/hakurei", acl.Execute).
					UpdatePerm("/tmp/hakurei.0/tmpdir/150", acl.Read, acl.Write, acl.Execute)
			}, []Op{
				&aclUpdateOp{Process, "/run/user/1971/hakurei", []acl.Perm{acl.Execute}},
				&aclUpdateOp{Process, "/tmp/hakurei.0/tmpdir/150", []acl.Perm{acl.Read, acl.Write, acl.Execute}},
			}, stub.Expect{}},

		{"tmpdirp", 0xdeadbeef, func(_ *testing.T, sys *I) {
			sys.UpdatePermType(User, "/tmp/hakurei.0/tmpdir", acl.Execute)
		}, []Op{
			&aclUpdateOp{User, "/tmp/hakurei.0/tmpdir", []acl.Perm{acl.Execute}},
		}, stub.Expect{}},

		{"tmpdir", 0xdeadbeef, func(_ *testing.T, sys *I) {
			sys.UpdatePermType(User, "/tmp/hakurei.0/tmpdir/150", acl.Read, acl.Write, acl.Execute)
		}, []Op{
			&aclUpdateOp{User, "/tmp/hakurei.0/tmpdir/150", []acl.Perm{acl.Read, acl.Write, acl.Execute}},
		}, stub.Expect{}},

		{"share", 0xdeadbeef, func(_ *testing.T, sys *I) {
			sys.UpdatePermType(Process, "/run/user/1971/hakurei/fcb8a12f7c482d183ade8288c3de78b5", acl.Execute)
		}, []Op{
			&aclUpdateOp{Process, "/run/user/1971/hakurei/fcb8a12f7c482d183ade8288c3de78b5", []acl.Perm{acl.Execute}},
		}, stub.Expect{}},

		{"passwd", 0xdeadbeef, func(_ *testing.T, sys *I) {
			sys.
				UpdatePermType(Process, "/tmp/hakurei.0/fcb8a12f7c482d183ade8288c3de78b5/passwd", acl.Read).
				UpdatePermType(Process, "/tmp/hakurei.0/fcb8a12f7c482d183ade8288c3de78b5/group", acl.Read)
		}, []Op{
			&aclUpdateOp{Process, "/tmp/hakurei.0/fcb8a12f7c482d183ade8288c3de78b5/passwd", []acl.Perm{acl.Read}},
			&aclUpdateOp{Process, "/tmp/hakurei.0/fcb8a12f7c482d183ade8288c3de78b5/group", []acl.Perm{acl.Read}},
		}, stub.Expect{}},

		{"wayland", 0xdeadbeef, func(_ *testing.T, sys *I) {
			sys.UpdatePermType(hst.EWayland, "/run/user/1971/wayland-0", acl.Read, acl.Write, acl.Execute)
		}, []Op{
			&aclUpdateOp{hst.EWayland, "/run/user/1971/wayland-0", []acl.Perm{acl.Read, acl.Write, acl.Execute}},
		}, stub.Expect{}},
	})

	checkOpIs(t, []opIsTestCase{
		{"nil", (*aclUpdateOp)(nil), (*aclUpdateOp)(nil), false},
		{"zero", new(aclUpdateOp), new(aclUpdateOp), true},

		{"et differs",
			&aclUpdateOp{
				hst.EWayland, "/run/user/1971/wayland-0",
				[]acl.Perm{acl.Read, acl.Write, acl.Execute},
			}, &aclUpdateOp{
				hst.EX11, "/run/user/1971/wayland-0",
				[]acl.Perm{acl.Read, acl.Write, acl.Execute},
			}, false},

		{"path differs", &aclUpdateOp{
			hst.EWayland, "/run/user/1971/wayland-0",
			[]acl.Perm{acl.Read, acl.Write, acl.Execute},
		}, &aclUpdateOp{
			hst.EWayland, "/run/user/1971/wayland-1",
			[]acl.Perm{acl.Read, acl.Write, acl.Execute},
		}, false},

		{"perms differs", &aclUpdateOp{
			hst.EWayland, "/run/user/1971/wayland-0",
			[]acl.Perm{acl.Read, acl.Write, acl.Execute},
		}, &aclUpdateOp{
			hst.EWayland, "/run/user/1971/wayland-0",
			[]acl.Perm{acl.Read, acl.Write},
		}, false},

		{"equals", &aclUpdateOp{
			hst.EWayland, "/run/user/1971/wayland-0",
			[]acl.Perm{acl.Read, acl.Write, acl.Execute},
		}, &aclUpdateOp{
			hst.EWayland, "/run/user/1971/wayland-0",
			[]acl.Perm{acl.Read, acl.Write, acl.Execute},
		}, true},
	})

	checkOpMeta(t, []opMetaTestCase{
		{"clear",
			&aclUpdateOp{Process, "/proc/nonexistent", []acl.Perm{}},
			Process, "/proc/nonexistent",
			`--- type: process path: "/proc/nonexistent"`},

		{"read",
			&aclUpdateOp{User, "/tmp/hakurei.0/27d81d567f8fae7f33278eec45da9446/0", []acl.Perm{acl.Read}},
			User, "/tmp/hakurei.0/27d81d567f8fae7f33278eec45da9446/0",
			`r-- type: user path: "/tmp/hakurei.0/27d81d567f8fae7f33278eec45da9446/0"`},

		{"write",
			&aclUpdateOp{User, "/tmp/hakurei.0/27d81d567f8fae7f33278eec45da9446/1", []acl.Perm{acl.Write}},
			User, "/tmp/hakurei.0/27d81d567f8fae7f33278eec45da9446/1",
			`-w- type: user path: "/tmp/hakurei.0/27d81d567f8fae7f33278eec45da9446/1"`},

		{"execute",
			&aclUpdateOp{User, "/tmp/hakurei.0/27d81d567f8fae7f33278eec45da9446/2", []acl.Perm{acl.Execute}},
			User, "/tmp/hakurei.0/27d81d567f8fae7f33278eec45da9446/2",
			`--x type: user path: "/tmp/hakurei.0/27d81d567f8fae7f33278eec45da9446/2"`},

		{"wayland",
			&aclUpdateOp{hst.EWayland, "/tmp/hakurei.0/27d81d567f8fae7f33278eec45da9446/wayland", []acl.Perm{acl.Read, acl.Write}},
			hst.EWayland, "/tmp/hakurei.0/27d81d567f8fae7f33278eec45da9446/wayland",
			`rw- type: wayland path: "/tmp/hakurei.0/27d81d567f8fae7f33278eec45da9446/wayland"`},

		{"x11",
			&aclUpdateOp{hst.EX11, "/tmp/.X11-unix/X0", []acl.Perm{acl.Read, acl.Execute}},
			hst.EX11, "/tmp/.X11-unix/X0",
			`r-x type: x11 path: "/tmp/.X11-unix/X0"`},

		{"dbus",
			&aclUpdateOp{hst.EDBus, "/tmp/hakurei.0/27d81d567f8fae7f33278eec45da9446/bus", []acl.Perm{acl.Write, acl.Execute}},
			hst.EDBus, "/tmp/hakurei.0/27d81d567f8fae7f33278eec45da9446/bus",
			`-wx type: dbus path: "/tmp/hakurei.0/27d81d567f8fae7f33278eec45da9446/bus"`},

		{"pulseaudio",
			&aclUpdateOp{hst.EPulse, "/run/user/1971/hakurei/27d81d567f8fae7f33278eec45da9446/pulse", []acl.Perm{acl.Read, acl.Write, acl.Execute}},
			hst.EPulse, "/run/user/1971/hakurei/27d81d567f8fae7f33278eec45da9446/pulse",
			`rwx type: pulseaudio path: "/run/user/1971/hakurei/27d81d567f8fae7f33278eec45da9446/pulse"`},
	})
}
