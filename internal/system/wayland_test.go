package system

import (
	"errors"
	"os"
	"testing"

	"hakurei.app/container/stub"
	"hakurei.app/internal/acl"
	"hakurei.app/system/wayland"
)

type stubWaylandConn struct {
	t *testing.T

	wantAttach string
	attachErr  error
	attached   bool

	wantBind [3]string
	bindErr  error
	bound    bool

	closeErr error
	closed   bool
}

func (conn *stubWaylandConn) Attach(p string) (err error) {
	conn.t.Helper()

	if conn.attached {
		conn.t.Fatal("Attach called twice")
	}
	conn.attached = true

	err = conn.attachErr
	if p != conn.wantAttach {
		conn.t.Errorf("Attach: p = %q, want %q", p, conn.wantAttach)
		err = stub.ErrCheck
	}
	return
}

func (conn *stubWaylandConn) Bind(pathname, appID, instanceID string) (*os.File, error) {
	conn.t.Helper()

	if !conn.attached {
		conn.t.Fatal("Bind called before Attach")
	}

	if conn.bound {
		conn.t.Fatal("Bind called twice")
	}
	conn.bound = true

	if pathname != conn.wantBind[0] {
		conn.t.Errorf("Attach: pathname = %q, want %q", pathname, conn.wantBind[0])
		return nil, stub.ErrCheck
	}
	if appID != conn.wantBind[1] {
		conn.t.Errorf("Attach: appID = %q, want %q", appID, conn.wantBind[1])
		return nil, stub.ErrCheck
	}
	if instanceID != conn.wantBind[2] {
		conn.t.Errorf("Attach: instanceID = %q, want %q", instanceID, conn.wantBind[2])
		return nil, stub.ErrCheck
	}
	return nil, conn.bindErr
}

func (conn *stubWaylandConn) Close() error {
	conn.t.Helper()

	if !conn.attached {
		conn.t.Fatal("Close called before Attach")
	}
	if !conn.bound {
		conn.t.Fatal("Close called before Bind")
	}

	if conn.closed {
		conn.t.Fatal("Close called twice")
	}
	conn.closed = true
	return conn.closeErr
}

func TestWaylandOp(t *testing.T) {
	t.Parallel()

	checkOpBehaviour(t, []opBehaviourTestCase{
		{"attach", 0xbeef, 0xff, &waylandOp{nil,
			"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland",
			"/run/user/1971/wayland-0",
			"org.chromium.Chromium",
			"ebf083d1b175911782d413369b64ce7c",
			&stubWaylandConn{t: t, wantAttach: "/run/user/1971/wayland-0", wantBind: [3]string{
				"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland",
				"org.chromium.Chromium", "ebf083d1b175911782d413369b64ce7c"},
				attachErr: stub.UniqueError(5)},
		}, nil, &OpError{Op: "wayland", Err: stub.UniqueError(5)}, nil, nil},

		{"bind", 0xbeef, 0xff, &waylandOp{nil,
			"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland",
			"/run/user/1971/wayland-0",
			"org.chromium.Chromium",
			"ebf083d1b175911782d413369b64ce7c",
			&stubWaylandConn{t: t, wantAttach: "/run/user/1971/wayland-0", wantBind: [3]string{
				"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland",
				"org.chromium.Chromium", "ebf083d1b175911782d413369b64ce7c"},
				bindErr: stub.UniqueError(4)},
		}, []stub.Call{
			call("verbosef", stub.ExpectArgs{"wayland attached on %q", []any{"/run/user/1971/wayland-0"}}, nil, nil),
		}, &OpError{Op: "wayland", Err: stub.UniqueError(4)}, nil, nil},

		{"chmod", 0xbeef, 0xff, &waylandOp{nil,
			"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland",
			"/run/user/1971/wayland-0",
			"org.chromium.Chromium",
			"ebf083d1b175911782d413369b64ce7c",
			&stubWaylandConn{t: t, wantAttach: "/run/user/1971/wayland-0", wantBind: [3]string{
				"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland",
				"org.chromium.Chromium", "ebf083d1b175911782d413369b64ce7c"}},
		}, []stub.Call{
			call("verbosef", stub.ExpectArgs{"wayland attached on %q", []any{"/run/user/1971/wayland-0"}}, nil, nil),
			call("verbosef", stub.ExpectArgs{"wayland listening on %q", []any{"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"}}, nil, nil),
			call("chmod", stub.ExpectArgs{"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland", os.FileMode(0)}, nil, stub.UniqueError(3)),
		}, &OpError{Op: "wayland", Err: stub.UniqueError(3)}, nil, nil},

		{"aclUpdate", 0xbeef, 0xff, &waylandOp{nil,
			"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland",
			"/run/user/1971/wayland-0",
			"org.chromium.Chromium",
			"ebf083d1b175911782d413369b64ce7c",
			&stubWaylandConn{t: t, wantAttach: "/run/user/1971/wayland-0", wantBind: [3]string{
				"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland",
				"org.chromium.Chromium", "ebf083d1b175911782d413369b64ce7c"}},
		}, []stub.Call{
			call("verbosef", stub.ExpectArgs{"wayland attached on %q", []any{"/run/user/1971/wayland-0"}}, nil, nil),
			call("verbosef", stub.ExpectArgs{"wayland listening on %q", []any{"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"}}, nil, nil),
			call("chmod", stub.ExpectArgs{"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland", os.FileMode(0)}, nil, nil),
			call("aclUpdate", stub.ExpectArgs{"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland", 0xbeef, []acl.Perm{acl.Read, acl.Write, acl.Execute}}, nil, stub.UniqueError(2)),
		}, &OpError{Op: "wayland", Err: stub.UniqueError(2)}, nil, nil},

		{"remove", 0xbeef, 0xff, &waylandOp{nil,
			"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland",
			"/run/user/1971/wayland-0",
			"org.chromium.Chromium",
			"ebf083d1b175911782d413369b64ce7c",
			&stubWaylandConn{t: t, wantAttach: "/run/user/1971/wayland-0", wantBind: [3]string{
				"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland",
				"org.chromium.Chromium", "ebf083d1b175911782d413369b64ce7c"}},
		}, []stub.Call{
			call("verbosef", stub.ExpectArgs{"wayland attached on %q", []any{"/run/user/1971/wayland-0"}}, nil, nil),
			call("verbosef", stub.ExpectArgs{"wayland listening on %q", []any{"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"}}, nil, nil),
			call("chmod", stub.ExpectArgs{"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland", os.FileMode(0)}, nil, nil),
			call("aclUpdate", stub.ExpectArgs{"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland", 0xbeef, []acl.Perm{acl.Read, acl.Write, acl.Execute}}, nil, nil),
		}, nil, []stub.Call{
			call("verbosef", stub.ExpectArgs{"detaching from wayland on %q", []any{"/run/user/1971/wayland-0"}}, nil, nil),
			call("verbosef", stub.ExpectArgs{"removing wayland socket on %q", []any{"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"}}, nil, nil),
			call("remove", stub.ExpectArgs{"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"}, nil, stub.UniqueError(1)),
		}, &OpError{Op: "wayland", Err: errors.Join(stub.UniqueError(1)), Revert: true}},

		{"close", 0xbeef, 0xff, &waylandOp{nil,
			"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland",
			"/run/user/1971/wayland-0",
			"org.chromium.Chromium",
			"ebf083d1b175911782d413369b64ce7c",
			&stubWaylandConn{t: t, wantAttach: "/run/user/1971/wayland-0", wantBind: [3]string{
				"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland",
				"org.chromium.Chromium", "ebf083d1b175911782d413369b64ce7c"},
				closeErr: stub.UniqueError(0)},
		}, []stub.Call{
			call("verbosef", stub.ExpectArgs{"wayland attached on %q", []any{"/run/user/1971/wayland-0"}}, nil, nil),
			call("verbosef", stub.ExpectArgs{"wayland listening on %q", []any{"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"}}, nil, nil),
			call("chmod", stub.ExpectArgs{"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland", os.FileMode(0)}, nil, nil),
			call("aclUpdate", stub.ExpectArgs{"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland", 0xbeef, []acl.Perm{acl.Read, acl.Write, acl.Execute}}, nil, nil),
		}, nil, []stub.Call{
			call("verbosef", stub.ExpectArgs{"detaching from wayland on %q", []any{"/run/user/1971/wayland-0"}}, nil, nil),
			call("verbosef", stub.ExpectArgs{"removing wayland socket on %q", []any{"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"}}, nil, nil),
			call("remove", stub.ExpectArgs{"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"}, nil, nil),
		}, &OpError{Op: "wayland", Err: errors.Join(stub.UniqueError(0)), Revert: true}},

		{"success", 0xbeef, 0xff, &waylandOp{nil,
			"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland",
			"/run/user/1971/wayland-0",
			"org.chromium.Chromium",
			"ebf083d1b175911782d413369b64ce7c",
			&stubWaylandConn{t: t, wantAttach: "/run/user/1971/wayland-0", wantBind: [3]string{
				"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland",
				"org.chromium.Chromium", "ebf083d1b175911782d413369b64ce7c"}},
		}, []stub.Call{
			call("verbosef", stub.ExpectArgs{"wayland attached on %q", []any{"/run/user/1971/wayland-0"}}, nil, nil),
			call("verbosef", stub.ExpectArgs{"wayland listening on %q", []any{"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"}}, nil, nil),
			call("chmod", stub.ExpectArgs{"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland", os.FileMode(0)}, nil, nil),
			call("aclUpdate", stub.ExpectArgs{"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland", 0xbeef, []acl.Perm{acl.Read, acl.Write, acl.Execute}}, nil, nil),
		}, nil, []stub.Call{
			call("verbosef", stub.ExpectArgs{"detaching from wayland on %q", []any{"/run/user/1971/wayland-0"}}, nil, nil),
			call("verbosef", stub.ExpectArgs{"removing wayland socket on %q", []any{"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"}}, nil, nil),
			call("remove", stub.ExpectArgs{"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"}, nil, nil),
		}, nil},
	})

	checkOpsBuilder(t, "Wayland", []opsBuilderTestCase{
		{"chromium", 0xcafe, func(_ *testing.T, sys *I) {
			sys.Wayland(
				m("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"),
				m("/run/user/1971/wayland-0"),
				"org.chromium.Chromium",
				"ebf083d1b175911782d413369b64ce7c",
			)
		}, []Op{&waylandOp{nil,
			"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland",
			"/run/user/1971/wayland-0",
			"org.chromium.Chromium",
			"ebf083d1b175911782d413369b64ce7c",
			new(wayland.Conn),
		}}, stub.Expect{}},
	})

	checkOpIs(t, []opIsTestCase{
		{"dst differs", &waylandOp{nil,
			"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7d/wayland",
			"/run/user/1971/wayland-0",
			"org.chromium.Chromium",
			"ebf083d1b175911782d413369b64ce7c",
			new(wayland.Conn),
		}, &waylandOp{nil,
			"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland",
			"/run/user/1971/wayland-0",
			"org.chromium.Chromium",
			"ebf083d1b175911782d413369b64ce7c",
			new(wayland.Conn),
		}, false},

		{"src differs", &waylandOp{nil,
			"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland",
			"/run/user/1971/wayland-1",
			"org.chromium.Chromium",
			"ebf083d1b175911782d413369b64ce7c",
			new(wayland.Conn),
		}, &waylandOp{nil,
			"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland",
			"/run/user/1971/wayland-0",
			"org.chromium.Chromium",
			"ebf083d1b175911782d413369b64ce7c",
			new(wayland.Conn),
		}, false},

		{"appID differs", &waylandOp{nil,
			"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland",
			"/run/user/1971/wayland-0",
			"org.chromium",
			"ebf083d1b175911782d413369b64ce7c",
			new(wayland.Conn),
		}, &waylandOp{nil,
			"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland",
			"/run/user/1971/wayland-0",
			"org.chromium.Chromium",
			"ebf083d1b175911782d413369b64ce7c",
			new(wayland.Conn),
		}, false},

		{"instanceID differs", &waylandOp{nil,
			"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland",
			"/run/user/1971/wayland-0",
			"org.chromium.Chromium",
			"ebf083d1b175911782d413369b64ce7d",
			new(wayland.Conn),
		}, &waylandOp{nil,
			"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland",
			"/run/user/1971/wayland-0",
			"org.chromium.Chromium",
			"ebf083d1b175911782d413369b64ce7c",
			new(wayland.Conn),
		}, false},

		{"equals", &waylandOp{nil,
			"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland",
			"/run/user/1971/wayland-0",
			"org.chromium.Chromium",
			"ebf083d1b175911782d413369b64ce7c",
			new(wayland.Conn),
		}, &waylandOp{nil,
			"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland",
			"/run/user/1971/wayland-0",
			"org.chromium.Chromium",
			"ebf083d1b175911782d413369b64ce7c",
			new(wayland.Conn),
		}, true},
	})

	checkOpMeta(t, []opMetaTestCase{
		{"chromium", &waylandOp{nil,
			"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland",
			"/run/user/1971/wayland-0",
			"org.chromium.Chromium",
			"ebf083d1b175911782d413369b64ce7c",
			new(wayland.Conn),
		}, Process, "/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland",
			`wayland socket at "/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"`},
	})
}
