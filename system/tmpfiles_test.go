package system

import (
	"strconv"
	"testing"
)

func TestCopyFile(t *testing.T) {
	testCases := []struct {
		tcOp
		cap int
		n   int64
	}{
		{tcOp{Process, "/home/ophestra/xdg/config/pulse/cookie"}, 256, 256},
	}
	for _, tc := range testCases {
		t.Run("copy file "+tc.path+" with cap = "+strconv.Itoa(tc.cap)+" n = "+strconv.Itoa(int(tc.n)), func(t *testing.T) {
			sys := New(150)
			sys.CopyFile(new([]byte), tc.path, tc.cap, tc.n)
			tc.test(t, sys.ops, []Op{
				&Tmpfile{nil, tc.path, tc.n, nil},
			}, "CopyFile")
		})
	}
}

func TestLink(t *testing.T) {
	testCases := []struct {
		dst, src string
	}{
		{"/tmp/hakurei.1971/f587afe9fce3c8e1ad5b64deb6c41ad5/pulse-cookie", "/home/ophestra/xdg/config/pulse/cookie"},
		{"/tmp/hakurei.1971/62154f708b5184ab01f9dcc2bbe7a33b/pulse-cookie", "/home/ophestra/xdg/config/pulse/cookie"},
	}
	for _, tc := range testCases {
		t.Run("link file "+tc.dst+" from "+tc.src, func(t *testing.T) {
			sys := New(150)
			sys.Link(tc.src, tc.dst)
			(&tcOp{Process, tc.src}).test(t, sys.ops, []Op{
				&Hardlink{Process, tc.dst, tc.src},
			}, "Link")
		})
	}
}

func TestLinkFileType(t *testing.T) {
	testCases := []struct {
		tcOp
		dst string
	}{
		{tcOp{User, "/tmp/hakurei.1971/f587afe9fce3c8e1ad5b64deb6c41ad5/pulse-cookie"}, "/home/ophestra/xdg/config/pulse/cookie"},
		{tcOp{Process, "/tmp/hakurei.1971/62154f708b5184ab01f9dcc2bbe7a33b/pulse-cookie"}, "/home/ophestra/xdg/config/pulse/cookie"},
	}
	for _, tc := range testCases {
		t.Run("link file "+tc.dst+" from "+tc.path+" with type "+TypeString(tc.et), func(t *testing.T) {
			sys := New(150)
			sys.LinkFileType(tc.et, tc.path, tc.dst)
			tc.test(t, sys.ops, []Op{
				&Hardlink{tc.et, tc.dst, tc.path},
			}, "LinkFileType")
		})
	}
}

func TestTmpfile_String(t *testing.T) {
	testCases := []struct {
		src  string
		n    int64
		want string
	}{
		{"/home/ophestra/xdg/config/pulse/cookie", 256,
			`up to 256 bytes from "/home/ophestra/xdg/config/pulse/cookie"`},
	}

	for _, tc := range testCases {
		t.Run(tc.want, func(t *testing.T) {
			if got := (&Tmpfile{src: tc.src, n: tc.n}).String(); got != tc.want {
				t.Errorf("String() = %v, want %v", got, tc.want)
			}
		})
	}
}
