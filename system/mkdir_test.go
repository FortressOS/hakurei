package system

import (
	"os"
	"testing"

	"hakurei.app/container/stub"
)

func TestMkdirOp(t *testing.T) {
	checkOpBehaviour(t, []opBehaviourTestCase{
		{"mkdir", 0xdeadbeef, 0xff, &mkdirOp{User, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0711, false}, []stub.Call{
			call("verbose", stub.ExpectArgs{[]any{"ensuring directory", &mkdirOp{User, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0711, false}}}, nil, nil),
			call("mkdir", stub.ExpectArgs{"/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", os.FileMode(0711)}, nil, stub.UniqueError(2)),
		}, &OpError{Op: "mkdir", Err: stub.UniqueError(2)}, nil, nil},

		{"chmod", 0xdeadbeef, 0xff, &mkdirOp{User, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0711, false}, []stub.Call{
			call("verbose", stub.ExpectArgs{[]any{"ensuring directory", &mkdirOp{User, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0711, false}}}, nil, nil),
			call("mkdir", stub.ExpectArgs{"/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", os.FileMode(0711)}, nil, os.ErrExist),
			call("chmod", stub.ExpectArgs{"/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", os.FileMode(0711)}, nil, stub.UniqueError(1)),
		}, &OpError{Op: "mkdir", Err: stub.UniqueError(1)}, nil, nil},

		{"remove", 0xdeadbeef, 0xff, &mkdirOp{Process, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0711, true}, []stub.Call{
			call("verbose", stub.ExpectArgs{[]any{"ensuring directory", &mkdirOp{Process, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0711, true}}}, nil, nil),
			call("mkdir", stub.ExpectArgs{"/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", os.FileMode(0711)}, nil, nil),
		}, nil, []stub.Call{
			call("verbose", stub.ExpectArgs{[]any{"destroying ephemeral directory", &mkdirOp{Process, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0711, true}}}, nil, nil),
			call("remove", stub.ExpectArgs{"/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9"}, nil, stub.UniqueError(0)),
		}, &OpError{Op: "mkdir", Err: stub.UniqueError(0), Revert: true}},

		{"success exist chmod", 0xdeadbeef, 0xff, &mkdirOp{User, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0711, false}, []stub.Call{
			call("verbose", stub.ExpectArgs{[]any{"ensuring directory", &mkdirOp{User, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0711, false}}}, nil, nil),
			call("mkdir", stub.ExpectArgs{"/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", os.FileMode(0711)}, nil, os.ErrExist),
			call("chmod", stub.ExpectArgs{"/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", os.FileMode(0711)}, nil, nil),
		}, nil, nil, nil},

		{"success ensure", 0xdeadbeef, 0xff, &mkdirOp{User, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0711, false}, []stub.Call{
			call("verbose", stub.ExpectArgs{[]any{"ensuring directory", &mkdirOp{User, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0711, false}}}, nil, nil),
			call("mkdir", stub.ExpectArgs{"/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", os.FileMode(0711)}, nil, nil),
		}, nil, nil, nil},

		{"success skip", 0xdeadbeef, 0xff, &mkdirOp{User, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0711, true}, []stub.Call{
			call("verbose", stub.ExpectArgs{[]any{"ensuring directory", &mkdirOp{User, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0711, true}}}, nil, nil),
			call("mkdir", stub.ExpectArgs{"/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", os.FileMode(0711)}, nil, nil),
		}, nil, []stub.Call{
			call("verbose", stub.ExpectArgs{[]any{"skipping ephemeral directory", &mkdirOp{User, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0711, true}}}, nil, nil),
		}, nil},

		{"success", 0xdeadbeef, 0xff, &mkdirOp{Process, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0711, true}, []stub.Call{
			call("verbose", stub.ExpectArgs{[]any{"ensuring directory", &mkdirOp{Process, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0711, true}}}, nil, nil),
			call("mkdir", stub.ExpectArgs{"/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", os.FileMode(0711)}, nil, nil),
		}, nil, []stub.Call{
			call("verbose", stub.ExpectArgs{[]any{"destroying ephemeral directory", &mkdirOp{Process, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0711, true}}}, nil, nil),
			call("remove", stub.ExpectArgs{"/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9"}, nil, nil),
		}, nil},
	})

	checkOpsBuilder(t, "EnsureEphemeral", []opsBuilderTestCase{
		{"ensure", 0xcafebabe, func(_ *testing.T, sys *I) {
			sys.Ensure("/tmp/hakurei.0", 0700)
		}, []Op{
			&mkdirOp{User, "/tmp/hakurei.0", 0700, false},
		}, stub.Expect{}},

		{"ephemeral", 0xcafebabe, func(_ *testing.T, sys *I) {
			sys.Ephemeral(Process, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0711)
		}, []Op{
			&mkdirOp{Process, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0711, true},
		}, stub.Expect{}},
	})

	checkOpIs(t, []opIsTestCase{
		{"nil", (*mkdirOp)(nil), (*mkdirOp)(nil), false},
		{"zero", new(mkdirOp), new(mkdirOp), true},

		{"ephemeral differs",
			&mkdirOp{Process, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0700, false},
			&mkdirOp{Process, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0700, true},
			false},

		{"perm differs",
			&mkdirOp{Process, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0701, true},
			&mkdirOp{Process, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0700, true},
			false},

		{"path differs",
			&mkdirOp{Process, "/tmp/hakurei.1/f2f3bcd492d0266438fa9bf164fe90d9", 0700, true},
			&mkdirOp{Process, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0700, true},
			false},

		{"et differs",
			&mkdirOp{User, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0700, true},
			&mkdirOp{Process, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0700, true},
			false},

		{"equals",
			&mkdirOp{Process, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0700, true},
			&mkdirOp{Process, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0700, true},
			true},
	})

	checkOpMeta(t, []opMetaTestCase{
		{"ensure", &mkdirOp{User, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0700, false},
			User, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9",
			`mode: -rwx------ type: ensure path: "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9"`},

		{"ephemeral", &mkdirOp{User, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0700, true},
			User, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9",
			`mode: -rwx------ type: user path: "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9"`},
	})
}
