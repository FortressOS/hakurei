package system

import (
	"errors"
	"os"
	"testing"

	"hakurei.app/container/stub"
	"hakurei.app/internal/acl"
)

func TestWaylandOp(t *testing.T) {
	t.Parallel()

	checkOpBehaviour(t, []opBehaviourTestCase{
		{"chmod", 0xbeef, 0xff, &waylandOp{nil,
			m("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"),
			m("/run/user/1971/wayland-0"),
			"org.chromium.Chromium",
			"ebf083d1b175911782d413369b64ce7c",
		}, []stub.Call{
			call("waylandNew", stub.ExpectArgs{m("/run/user/1971/wayland-0"), m("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"), "org.chromium.Chromium", "ebf083d1b175911782d413369b64ce7c"}, nil, nil),
			call("verbosef", stub.ExpectArgs{"wayland pathname socket on %q via %q", []any{m("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"), m("/run/user/1971/wayland-0")}}, nil, nil),
			call("chmod", stub.ExpectArgs{"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland", os.FileMode(0)}, nil, stub.UniqueError(3)),
		}, &OpError{Op: "wayland", Err: errors.Join(stub.UniqueError(3), os.ErrInvalid)}, nil, nil},

		{"aclUpdate", 0xbeef, 0xff, &waylandOp{nil,
			m("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"),
			m("/run/user/1971/wayland-0"),
			"org.chromium.Chromium",
			"ebf083d1b175911782d413369b64ce7c",
		}, []stub.Call{
			call("waylandNew", stub.ExpectArgs{m("/run/user/1971/wayland-0"), m("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"), "org.chromium.Chromium", "ebf083d1b175911782d413369b64ce7c"}, nil, nil),
			call("verbosef", stub.ExpectArgs{"wayland pathname socket on %q via %q", []any{m("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"), m("/run/user/1971/wayland-0")}}, nil, nil),
			call("chmod", stub.ExpectArgs{"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland", os.FileMode(0)}, nil, nil),
			call("aclUpdate", stub.ExpectArgs{"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland", 0xbeef, []acl.Perm{acl.Read, acl.Write, acl.Execute}}, nil, stub.UniqueError(2)),
		}, &OpError{Op: "wayland", Err: errors.Join(stub.UniqueError(2), os.ErrInvalid)}, nil, nil},

		{"remove", 0xbeef, 0xff, &waylandOp{nil,
			m("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"),
			m("/run/user/1971/wayland-0"),
			"org.chromium.Chromium",
			"ebf083d1b175911782d413369b64ce7c",
		}, []stub.Call{
			call("waylandNew", stub.ExpectArgs{m("/run/user/1971/wayland-0"), m("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"), "org.chromium.Chromium", "ebf083d1b175911782d413369b64ce7c"}, nil, nil),
			call("verbosef", stub.ExpectArgs{"wayland pathname socket on %q via %q", []any{m("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"), m("/run/user/1971/wayland-0")}}, nil, nil),
			call("chmod", stub.ExpectArgs{"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland", os.FileMode(0)}, nil, nil),
			call("aclUpdate", stub.ExpectArgs{"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland", 0xbeef, []acl.Perm{acl.Read, acl.Write, acl.Execute}}, nil, nil),
		}, nil, []stub.Call{
			call("verbosef", stub.ExpectArgs{"hanging up wayland socket on %q", []any{m("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland")}}, nil, nil),
			call("remove", stub.ExpectArgs{"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"}, nil, stub.UniqueError(1)),
		}, &OpError{Op: "wayland", Err: errors.Join(stub.UniqueError(1)), Revert: true}},

		{"success", 0xbeef, 0xff, &waylandOp{nil,
			m("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"),
			m("/run/user/1971/wayland-0"),
			"org.chromium.Chromium",
			"ebf083d1b175911782d413369b64ce7c",
		}, []stub.Call{
			call("waylandNew", stub.ExpectArgs{m("/run/user/1971/wayland-0"), m("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"), "org.chromium.Chromium", "ebf083d1b175911782d413369b64ce7c"}, nil, nil),
			call("verbosef", stub.ExpectArgs{"wayland pathname socket on %q via %q", []any{m("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"), m("/run/user/1971/wayland-0")}}, nil, nil),
			call("chmod", stub.ExpectArgs{"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland", os.FileMode(0)}, nil, nil),
			call("aclUpdate", stub.ExpectArgs{"/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland", 0xbeef, []acl.Perm{acl.Read, acl.Write, acl.Execute}}, nil, nil),
		}, nil, []stub.Call{
			call("verbosef", stub.ExpectArgs{"hanging up wayland socket on %q", []any{m("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland")}}, nil, nil),
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
			m("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"),
			m("/run/user/1971/wayland-0"),
			"org.chromium.Chromium",
			"ebf083d1b175911782d413369b64ce7c",
		}}, stub.Expect{}},
	})

	checkOpIs(t, []opIsTestCase{
		{"dst differs", &waylandOp{nil,
			m("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7d/wayland"),
			m("/run/user/1971/wayland-0"),
			"org.chromium.Chromium",
			"ebf083d1b175911782d413369b64ce7c",
		}, &waylandOp{nil,
			m("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"),
			m("/run/user/1971/wayland-0"),
			"org.chromium.Chromium",
			"ebf083d1b175911782d413369b64ce7c",
		}, false},

		{"src differs", &waylandOp{nil,
			m("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"),
			m("/run/user/1971/wayland-1"),
			"org.chromium.Chromium",
			"ebf083d1b175911782d413369b64ce7c",
		}, &waylandOp{nil,
			m("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"),
			m("/run/user/1971/wayland-0"),
			"org.chromium.Chromium",
			"ebf083d1b175911782d413369b64ce7c",
		}, false},

		{"appID differs", &waylandOp{nil,
			m("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"),
			m("/run/user/1971/wayland-0"),
			"org.chromium",
			"ebf083d1b175911782d413369b64ce7c",
		}, &waylandOp{nil,
			m("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"),
			m("/run/user/1971/wayland-0"),
			"org.chromium.Chromium",
			"ebf083d1b175911782d413369b64ce7c",
		}, false},

		{"instanceID differs", &waylandOp{nil,
			m("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"),
			m("/run/user/1971/wayland-0"),
			"org.chromium.Chromium",
			"ebf083d1b175911782d413369b64ce7d",
		}, &waylandOp{nil,
			m("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"),
			m("/run/user/1971/wayland-0"),
			"org.chromium.Chromium",
			"ebf083d1b175911782d413369b64ce7c",
		}, false},

		{"equals", &waylandOp{nil,
			m("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"),
			m("/run/user/1971/wayland-0"),
			"org.chromium.Chromium",
			"ebf083d1b175911782d413369b64ce7c",
		}, &waylandOp{nil,
			m("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"),
			m("/run/user/1971/wayland-0"),
			"org.chromium.Chromium",
			"ebf083d1b175911782d413369b64ce7c",
		}, true},
	})

	checkOpMeta(t, []opMetaTestCase{
		{"chromium", &waylandOp{nil,
			m("/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"),
			m("/run/user/1971/wayland-0"),
			"org.chromium.Chromium",
			"ebf083d1b175911782d413369b64ce7c",
		}, Process, "/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland",
			`wayland socket at "/tmp/hakurei.1971/ebf083d1b175911782d413369b64ce7c/wayland"`},
	})
}
