package system

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"syscall"
	"testing"

	"hakurei.app/container/stub"
)

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, stub.UniqueError(0xdeadbeef) }

func TestTmpfileOp(t *testing.T) {
	// 255 bytes
	const paSample = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

	checkOpBehaviour(t, []opBehaviourTestCase{
		{"payload", 0xdead, 0xff, &tmpfileOp{
			nil, "/home/ophestra/xdg/config/pulse/cookie", 1 << 8,
			func() *bytes.Buffer { buf := new(bytes.Buffer); buf.Grow(1 << 8); return buf }(),
		}, nil, errors.New("invalid payload"), nil, nil},

		{"stat", 0xdead, 0xff, &tmpfileOp{
			new([]byte), "/home/ophestra/xdg/config/pulse/cookie", 1 << 8,
			func() *bytes.Buffer { buf := new(bytes.Buffer); buf.Grow(1 << 8); return buf }(),
		}, []stub.Call{
			call("verbose", stub.ExpectArgs{[]any{"copying", &tmpfileOp{new([]byte), "/home/ophestra/xdg/config/pulse/cookie", 1 << 8, func() *bytes.Buffer { buf := new(bytes.Buffer); buf.Grow(1 << 8); return buf }()}}}, nil, nil),
			call("stat", stub.ExpectArgs{"/home/ophestra/xdg/config/pulse/cookie"}, nil, stub.UniqueError(1)),
		}, &OpError{Op: "tmpfile", Err: stub.UniqueError(1)}, nil, nil},

		{"stat EISDIR", 0xdead, 0xff, &tmpfileOp{
			new([]byte), "/home/ophestra/xdg/config/pulse/cookie", 1 << 8,
			func() *bytes.Buffer { buf := new(bytes.Buffer); buf.Grow(1 << 8); return buf }(),
		}, []stub.Call{
			call("verbose", stub.ExpectArgs{[]any{"copying", &tmpfileOp{new([]byte), "/home/ophestra/xdg/config/pulse/cookie", 1 << 8, func() *bytes.Buffer { buf := new(bytes.Buffer); buf.Grow(1 << 8); return buf }()}}}, nil, nil),
			call("stat", stub.ExpectArgs{"/home/ophestra/xdg/config/pulse/cookie"}, stubFi{1 << 8, true}, nil),
		}, &OpError{Op: "tmpfile", Err: &os.PathError{Op: "stat", Path: "/home/ophestra/xdg/config/pulse/cookie", Err: syscall.EISDIR}}, nil, nil},

		{"stat ENOMEM", 0xdead, 0xff, &tmpfileOp{
			new([]byte), "/home/ophestra/xdg/config/pulse/cookie", 1 << 8,
			func() *bytes.Buffer { buf := new(bytes.Buffer); buf.Grow(1 << 8); return buf }(),
		}, []stub.Call{
			call("verbose", stub.ExpectArgs{[]any{"copying", &tmpfileOp{new([]byte), "/home/ophestra/xdg/config/pulse/cookie", 1 << 8, func() *bytes.Buffer { buf := new(bytes.Buffer); buf.Grow(1 << 8); return buf }()}}}, nil, nil),
			call("stat", stub.ExpectArgs{"/home/ophestra/xdg/config/pulse/cookie"}, stubFi{1<<8 + 1, false}, nil),
		}, &OpError{Op: "tmpfile", Err: &os.PathError{Op: "stat", Path: "/home/ophestra/xdg/config/pulse/cookie", Err: syscall.ENOMEM}}, nil, nil},

		{"open", 0xdead, 0xff, &tmpfileOp{
			new([]byte), "/home/ophestra/xdg/config/pulse/cookie", 1 << 8,
			func() *bytes.Buffer { buf := new(bytes.Buffer); buf.Grow(1 << 8); return buf }(),
		}, []stub.Call{
			call("verbose", stub.ExpectArgs{[]any{"copying", &tmpfileOp{new([]byte), "/home/ophestra/xdg/config/pulse/cookie", 1 << 8, func() *bytes.Buffer { buf := new(bytes.Buffer); buf.Grow(1 << 8); return buf }()}}}, nil, nil),
			call("stat", stub.ExpectArgs{"/home/ophestra/xdg/config/pulse/cookie"}, stubFi{1 << 8, false}, nil),
			call("open", stub.ExpectArgs{"/home/ophestra/xdg/config/pulse/cookie"}, nil, stub.UniqueError(0)),
		}, &OpError{Op: "tmpfile", Err: stub.UniqueError(0)}, nil, nil},

		{"reader", 0xdead, 0xff, &tmpfileOp{
			new([]byte), "/home/ophestra/xdg/config/pulse/cookie", 1 << 8,
			func() *bytes.Buffer { buf := new(bytes.Buffer); buf.Grow(1 << 8); return buf }(),
		}, []stub.Call{
			call("verbose", stub.ExpectArgs{[]any{"copying", &tmpfileOp{new([]byte), "/home/ophestra/xdg/config/pulse/cookie", 1 << 8, func() *bytes.Buffer { buf := new(bytes.Buffer); buf.Grow(1 << 8); return buf }()}}}, nil, nil),
			call("stat", stub.ExpectArgs{"/home/ophestra/xdg/config/pulse/cookie"}, stubFi{1 << 8, false}, nil),
			call("open", stub.ExpectArgs{"/home/ophestra/xdg/config/pulse/cookie"}, &readerOsFile{true, errorReader{}}, nil),
		}, &OpError{Op: "tmpfile", Err: stub.UniqueError(0xdeadbeef)}, nil, nil},

		{"closed", 0xdead, 0xff, &tmpfileOp{
			new([]byte), "/home/ophestra/xdg/config/pulse/cookie", 1 << 8,
			func() *bytes.Buffer { buf := new(bytes.Buffer); buf.Grow(1 << 8); return buf }(),
		}, []stub.Call{
			call("verbose", stub.ExpectArgs{[]any{"copying", &tmpfileOp{new([]byte), "/home/ophestra/xdg/config/pulse/cookie", 1 << 8, func() *bytes.Buffer { buf := new(bytes.Buffer); buf.Grow(1 << 8); return buf }()}}}, nil, nil),
			call("stat", stub.ExpectArgs{"/home/ophestra/xdg/config/pulse/cookie"}, stubFi{1 << 8, false}, nil),
			call("open", stub.ExpectArgs{"/home/ophestra/xdg/config/pulse/cookie"}, &readerOsFile{true, strings.NewReader(paSample + "=")}, nil),
		}, &OpError{Op: "tmpfile", Err: os.ErrClosed}, nil, nil},

		{"success full", 0xdead, 0xff, &tmpfileOp{
			new([]byte), "/home/ophestra/xdg/config/pulse/cookie", 1 << 8,
			func() *bytes.Buffer { buf := new(bytes.Buffer); buf.Grow(1 << 8); return buf }(),
		}, []stub.Call{
			call("verbose", stub.ExpectArgs{[]any{"copying", &tmpfileOp{new([]byte), "/home/ophestra/xdg/config/pulse/cookie", 1 << 8, func() *bytes.Buffer { buf := new(bytes.Buffer); buf.Grow(1 << 8); return buf }()}}}, nil, nil),
			call("stat", stub.ExpectArgs{"/home/ophestra/xdg/config/pulse/cookie"}, stubFi{1 << 8, false}, nil),
			call("open", stub.ExpectArgs{"/home/ophestra/xdg/config/pulse/cookie"}, &readerOsFile{false, strings.NewReader(paSample + "=")}, nil),
		}, nil, nil, nil},

		{"success", 0xdead, 0xff, &tmpfileOp{
			new([]byte), "/home/ophestra/xdg/config/pulse/cookie", 1 << 8,
			func() *bytes.Buffer { buf := new(bytes.Buffer); buf.Grow(1 << 8); return buf }(),
		}, []stub.Call{
			call("verbose", stub.ExpectArgs{[]any{"copying", &tmpfileOp{new([]byte), "/home/ophestra/xdg/config/pulse/cookie", 1 << 8, func() *bytes.Buffer { buf := new(bytes.Buffer); buf.Grow(1 << 8); return buf }()}}}, nil, nil),
			call("stat", stub.ExpectArgs{"/home/ophestra/xdg/config/pulse/cookie"}, stubFi{1 << 8, false}, nil),
			call("open", stub.ExpectArgs{"/home/ophestra/xdg/config/pulse/cookie"}, &readerOsFile{false, strings.NewReader(paSample)}, nil),
			call("verbosef", stub.ExpectArgs{"copied %d bytes from %q", []any{int64(1<<8 - 1), "/home/ophestra/xdg/config/pulse/cookie"}}, nil, nil),
		}, nil, nil, nil},
	})

	checkOpsBuilder(t, "CopyFile", []opsBuilderTestCase{
		{"pulse", 0xcafebabe, func(_ *testing.T, sys *I) {
			sys.CopyFile(new([]byte), m("/home/ophestra/xdg/config/pulse/cookie"), 1<<8, 1<<8)
		}, []Op{&tmpfileOp{
			new([]byte), "/home/ophestra/xdg/config/pulse/cookie", 1 << 8,
			func() *bytes.Buffer { buf := new(bytes.Buffer); buf.Grow(1 << 8); return buf }(),
		}}, stub.Expect{}},
	})

	checkOpIs(t, []opIsTestCase{
		{"nil", (*tmpfileOp)(nil), (*tmpfileOp)(nil), false},
		{"zero", new(tmpfileOp), new(tmpfileOp), true},

		{"n differs", &tmpfileOp{
			src: "/home/ophestra/xdg/config/pulse/cookie",
			n:   1 << 7,
		}, &tmpfileOp{
			src: "/home/ophestra/xdg/config/pulse/cookie",
			n:   1 << 8,
		}, false},

		{"src differs", &tmpfileOp{
			src: "/home/ophestra/xdg/config/pulse",
			n:   1 << 8,
		}, &tmpfileOp{
			src: "/home/ophestra/xdg/config/pulse/cookie",
			n:   1 << 8,
		}, false},

		{"equals", &tmpfileOp{
			src: "/home/ophestra/xdg/config/pulse/cookie",
			n:   1 << 8,
		}, &tmpfileOp{
			src: "/home/ophestra/xdg/config/pulse/cookie",
			n:   1 << 8,
		}, true},
	})

	checkOpMeta(t, []opMetaTestCase{
		{"pulse", &tmpfileOp{nil, "/home/ophestra/xdg/config/pulse/cookie", 1 << 8, nil},
			Process, "/home/ophestra/xdg/config/pulse/cookie",
			`up to 256 bytes from "/home/ophestra/xdg/config/pulse/cookie"`},
	})
}
