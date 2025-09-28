package system

import (
	"testing"

	"hakurei.app/container/stub"
	"hakurei.app/hst"
)

func TestHardlinkOp(t *testing.T) {
	checkOpBehaviour(t, []opBehaviourTestCase{
		{"link", 0xdeadbeef, 0xff, &hardlinkOp{hst.EPulse, "/run/user/1000/hakurei/9663730666a44cfc2a81610379e02ed6/pulse", "/run/user/1000/pulse/native"}, []stub.Call{
			call("verbose", stub.ExpectArgs{[]any{"linking", &hardlinkOp{hst.EPulse, "/run/user/1000/hakurei/9663730666a44cfc2a81610379e02ed6/pulse", "/run/user/1000/pulse/native"}}}, nil, nil),
			call("link", stub.ExpectArgs{"/run/user/1000/pulse/native", "/run/user/1000/hakurei/9663730666a44cfc2a81610379e02ed6/pulse"}, nil, stub.UniqueError(1)),
		}, &OpError{Op: "hardlink", Err: stub.UniqueError(1)}, nil, nil},

		{"remove", 0xdeadbeef, 0xff, &hardlinkOp{hst.EPulse, "/run/user/1000/hakurei/9663730666a44cfc2a81610379e02ed6/pulse", "/run/user/1000/pulse/native"}, []stub.Call{
			call("verbose", stub.ExpectArgs{[]any{"linking", &hardlinkOp{hst.EPulse, "/run/user/1000/hakurei/9663730666a44cfc2a81610379e02ed6/pulse", "/run/user/1000/pulse/native"}}}, nil, nil),
			call("link", stub.ExpectArgs{"/run/user/1000/pulse/native", "/run/user/1000/hakurei/9663730666a44cfc2a81610379e02ed6/pulse"}, nil, nil),
		}, nil, []stub.Call{
			call("verbosef", stub.ExpectArgs{"removing hard link %q", []any{"/run/user/1000/hakurei/9663730666a44cfc2a81610379e02ed6/pulse"}}, nil, nil),
			call("remove", stub.ExpectArgs{"/run/user/1000/hakurei/9663730666a44cfc2a81610379e02ed6/pulse"}, nil, stub.UniqueError(0)),
		}, &OpError{Op: "hardlink", Err: stub.UniqueError(0), Revert: true}},

		{"success skip", 0xdeadbeef, hst.EWayland | hst.EX11, &hardlinkOp{hst.EPulse, "/run/user/1000/hakurei/9663730666a44cfc2a81610379e02ed6/pulse", "/run/user/1000/pulse/native"}, []stub.Call{
			call("verbose", stub.ExpectArgs{[]any{"linking", &hardlinkOp{hst.EPulse, "/run/user/1000/hakurei/9663730666a44cfc2a81610379e02ed6/pulse", "/run/user/1000/pulse/native"}}}, nil, nil),
			call("link", stub.ExpectArgs{"/run/user/1000/pulse/native", "/run/user/1000/hakurei/9663730666a44cfc2a81610379e02ed6/pulse"}, nil, nil),
		}, nil, []stub.Call{
			call("verbosef", stub.ExpectArgs{"skipping hard link %q", []any{"/run/user/1000/hakurei/9663730666a44cfc2a81610379e02ed6/pulse"}}, nil, nil),
		}, nil},

		{"success", 0xdeadbeef, 0xff, &hardlinkOp{hst.EPulse, "/run/user/1000/hakurei/9663730666a44cfc2a81610379e02ed6/pulse", "/run/user/1000/pulse/native"}, []stub.Call{
			call("verbose", stub.ExpectArgs{[]any{"linking", &hardlinkOp{hst.EPulse, "/run/user/1000/hakurei/9663730666a44cfc2a81610379e02ed6/pulse", "/run/user/1000/pulse/native"}}}, nil, nil),
			call("link", stub.ExpectArgs{"/run/user/1000/pulse/native", "/run/user/1000/hakurei/9663730666a44cfc2a81610379e02ed6/pulse"}, nil, nil),
		}, nil, []stub.Call{
			call("verbosef", stub.ExpectArgs{"removing hard link %q", []any{"/run/user/1000/hakurei/9663730666a44cfc2a81610379e02ed6/pulse"}}, nil, nil),
			call("remove", stub.ExpectArgs{"/run/user/1000/hakurei/9663730666a44cfc2a81610379e02ed6/pulse"}, nil, nil),
		}, nil},
	})

	checkOpsBuilder(t, "LinkFileType", []opsBuilderTestCase{
		{"type", 0xcafebabe, func(_ *testing.T, sys *I) {
			sys.LinkFileType(User, "/run/user/1000/pulse/native", "/run/user/1000/hakurei/9663730666a44cfc2a81610379e02ed6/pulse")
		}, []Op{
			&hardlinkOp{User, "/run/user/1000/hakurei/9663730666a44cfc2a81610379e02ed6/pulse", "/run/user/1000/pulse/native"},
		}, stub.Expect{}},

		{"link", 0xcafebabe, func(_ *testing.T, sys *I) {
			sys.Link("/run/user/1000/pulse/native", "/run/user/1000/hakurei/9663730666a44cfc2a81610379e02ed6/pulse")
		}, []Op{
			&hardlinkOp{Process, "/run/user/1000/hakurei/9663730666a44cfc2a81610379e02ed6/pulse", "/run/user/1000/pulse/native"},
		}, stub.Expect{}},
	})

	checkOpIs(t, []opIsTestCase{
		{"nil", (*hardlinkOp)(nil), (*hardlinkOp)(nil), false},
		{"zero", new(hardlinkOp), new(hardlinkOp), true},

		{"src differs",
			&hardlinkOp{Process, "/run/user/1000/hakurei/9663730666a44cfc2a81610379e02ed6/pulse", "/run/user/1000/pulse"},
			&hardlinkOp{Process, "/run/user/1000/hakurei/9663730666a44cfc2a81610379e02ed6/pulse", "/run/user/1000/pulse/native"},
			false},

		{"dst differs",
			&hardlinkOp{Process, "/run/user/1000/hakurei/9663730666a44cfc2a81610379e02ed6", "/run/user/1000/pulse/native"},
			&hardlinkOp{Process, "/run/user/1000/hakurei/9663730666a44cfc2a81610379e02ed6/pulse", "/run/user/1000/pulse/native"},
			false},

		{"et differs",
			&hardlinkOp{User, "/run/user/1000/hakurei/9663730666a44cfc2a81610379e02ed6/pulse", "/run/user/1000/pulse/native"},
			&hardlinkOp{Process, "/run/user/1000/hakurei/9663730666a44cfc2a81610379e02ed6/pulse", "/run/user/1000/pulse/native"},
			false},

		{"equals",
			&hardlinkOp{Process, "/run/user/1000/hakurei/9663730666a44cfc2a81610379e02ed6/pulse", "/run/user/1000/pulse/native"},
			&hardlinkOp{Process, "/run/user/1000/hakurei/9663730666a44cfc2a81610379e02ed6/pulse", "/run/user/1000/pulse/native"},
			true},
	})

	checkOpMeta(t, []opMetaTestCase{
		{"link", &hardlinkOp{Process, "/run/user/1000/hakurei/9663730666a44cfc2a81610379e02ed6/pulse", "/run/user/1000/pulse/native"},
			Process, "/run/user/1000/pulse/native",
			`"/run/user/1000/hakurei/9663730666a44cfc2a81610379e02ed6/pulse" from "/run/user/1000/pulse/native"`},
	})
}
