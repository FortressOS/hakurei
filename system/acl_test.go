package system

import (
	"testing"

	"git.gensokyo.uk/security/fortify/acl"
)

func TestUpdatePerm(t *testing.T) {
	testCases := []struct {
		path  string
		perms []acl.Perm
	}{
		{"/run/user/1971/fortify", []acl.Perm{acl.Execute}},
		{"/tmp/fortify.1971/tmpdir/150", []acl.Perm{acl.Read, acl.Write, acl.Execute}},
	}

	for _, tc := range testCases {
		t.Run(tc.path+permSubTestSuffix(tc.perms), func(t *testing.T) {
			sys := New(150)
			sys.UpdatePerm(tc.path, tc.perms...)
			(&tcOp{Process, tc.path}).test(t, sys.ops, []Op{&ACL{Process, tc.path, tc.perms}}, "UpdatePerm")
		})
	}
}

func TestUpdatePermType(t *testing.T) {
	testCases := []struct {
		perms []acl.Perm
		tcOp
	}{
		{[]acl.Perm{acl.Execute}, tcOp{User, "/tmp/fortify.1971/tmpdir"}},
		{[]acl.Perm{acl.Read, acl.Write, acl.Execute}, tcOp{User, "/tmp/fortify.1971/tmpdir/150"}},
		{[]acl.Perm{acl.Execute}, tcOp{Process, "/run/user/1971/fortify/fcb8a12f7c482d183ade8288c3de78b5"}},
		{[]acl.Perm{acl.Read}, tcOp{Process, "/tmp/fortify.1971/fcb8a12f7c482d183ade8288c3de78b5/passwd"}},
		{[]acl.Perm{acl.Read}, tcOp{Process, "/tmp/fortify.1971/fcb8a12f7c482d183ade8288c3de78b5/group"}},
		{[]acl.Perm{acl.Read, acl.Write, acl.Execute}, tcOp{EWayland, "/run/user/1971/wayland-0"}},
	}

	for _, tc := range testCases {
		t.Run(tc.path+"_"+TypeString(tc.et)+permSubTestSuffix(tc.perms), func(t *testing.T) {
			sys := New(150)
			sys.UpdatePermType(tc.et, tc.path, tc.perms...)
			tc.test(t, sys.ops, []Op{&ACL{tc.et, tc.path, tc.perms}}, "UpdatePermType")
		})
	}
}

func TestACL_String(t *testing.T) {
	testCases := []struct {
		want  string
		et    Enablement
		perms []acl.Perm
	}{
		{`--- type: Process path: "/nonexistent"`, Process, []acl.Perm{}},
		{`r-- type: User path: "/nonexistent"`, User, []acl.Perm{acl.Read}},
		{`-w- type: Wayland path: "/nonexistent"`, EWayland, []acl.Perm{acl.Write}},
		{`--x type: X11 path: "/nonexistent"`, EX11, []acl.Perm{acl.Execute}},
		{`rw- type: D-Bus path: "/nonexistent"`, EDBus, []acl.Perm{acl.Read, acl.Write}},
		{`r-x type: PulseAudio path: "/nonexistent"`, EPulse, []acl.Perm{acl.Read, acl.Execute}},
		{`rwx type: User path: "/nonexistent"`, User, []acl.Perm{acl.Read, acl.Write, acl.Execute}},
		{`rwx type: Process path: "/nonexistent"`, Process, []acl.Perm{acl.Read, acl.Write, acl.Write, acl.Execute}},
	}

	for _, tc := range testCases {
		t.Run(tc.want, func(t *testing.T) {
			a := &ACL{et: tc.et, perms: tc.perms, path: "/nonexistent"}
			if got := a.String(); got != tc.want {
				t.Errorf("String() = %v, want %v",
					got, tc.want)
			}
		})
	}
}

func permSubTestSuffix(perms []acl.Perm) (suffix string) {
	for _, perm := range perms {
		switch perm {
		case acl.Read:
			suffix += "_read"
		case acl.Write:
			suffix += "_write"
		case acl.Execute:
			suffix += "_execute"
		default:
			panic("unreachable")
		}
	}
	return
}
