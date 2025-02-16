package system

import (
	"testing"

	"git.gensokyo.uk/security/fortify/acl"
)

func TestCopyFile(t *testing.T) {
	testCases := []struct {
		dst, src string
	}{
		{"/tmp/fortify.1971/f587afe9fce3c8e1ad5b64deb6c41ad5/pulse-cookie", "/home/ophestra/xdg/config/pulse/cookie"},
		{"/tmp/fortify.1971/62154f708b5184ab01f9dcc2bbe7a33b/pulse-cookie", "/home/ophestra/xdg/config/pulse/cookie"},
	}
	for _, tc := range testCases {
		t.Run("copy file "+tc.dst+" from "+tc.src, func(t *testing.T) {
			sys := New(150)
			sys.CopyFile(tc.dst, tc.src)
			(&tcOp{Process, tc.src}).test(t, sys.ops, []Op{
				&Tmpfile{Process, tmpfileCopy, tc.dst, tc.src},
				&ACL{Process, tc.dst, []acl.Perm{acl.Read}},
			}, "CopyFile")
		})
	}
}

func TestCopyFileType(t *testing.T) {
	testCases := []struct {
		tcOp
		dst string
	}{
		{tcOp{User, "/tmp/fortify.1971/f587afe9fce3c8e1ad5b64deb6c41ad5/pulse-cookie"}, "/home/ophestra/xdg/config/pulse/cookie"},
		{tcOp{Process, "/tmp/fortify.1971/62154f708b5184ab01f9dcc2bbe7a33b/pulse-cookie"}, "/home/ophestra/xdg/config/pulse/cookie"},
	}
	for _, tc := range testCases {
		t.Run("copy file "+tc.dst+" from "+tc.path+" with type "+TypeString(tc.et), func(t *testing.T) {
			sys := New(150)
			sys.CopyFileType(tc.et, tc.dst, tc.path)
			tc.test(t, sys.ops, []Op{
				&Tmpfile{tc.et, tmpfileCopy, tc.dst, tc.path},
				&ACL{tc.et, tc.dst, []acl.Perm{acl.Read}},
			}, "CopyFileType")
		})
	}
}

func TestLink(t *testing.T) {
	testCases := []struct {
		dst, src string
	}{
		{"/tmp/fortify.1971/f587afe9fce3c8e1ad5b64deb6c41ad5/pulse-cookie", "/home/ophestra/xdg/config/pulse/cookie"},
		{"/tmp/fortify.1971/62154f708b5184ab01f9dcc2bbe7a33b/pulse-cookie", "/home/ophestra/xdg/config/pulse/cookie"},
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
		{tcOp{User, "/tmp/fortify.1971/f587afe9fce3c8e1ad5b64deb6c41ad5/pulse-cookie"}, "/home/ophestra/xdg/config/pulse/cookie"},
		{tcOp{Process, "/tmp/fortify.1971/62154f708b5184ab01f9dcc2bbe7a33b/pulse-cookie"}, "/home/ophestra/xdg/config/pulse/cookie"},
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
	t.Run("invalid method panic", func(t *testing.T) {
		defer func() {
			wantPanic := "invalid tmpfile method 255"
			if r := recover(); r != wantPanic {
				t.Errorf("String() panic = %v, want %v",
					r, wantPanic)
			}
		}()
		_ = (&Tmpfile{method: 255}).String()
	})

	testCases := []struct {
		method   uint8
		dst, src string
		want     string
	}{
		{tmpfileCopy, "/tmp/fortify.1971/4b6bdc9182fb2f1d3a965c5fa8b9b66e/pulse-cookie", "/home/ophestra/xdg/config/pulse/cookie",
			`"/tmp/fortify.1971/4b6bdc9182fb2f1d3a965c5fa8b9b66e/pulse-cookie" from "/home/ophestra/xdg/config/pulse/cookie"`},
	}

	for _, tc := range testCases {
		t.Run(tc.want, func(t *testing.T) {
			if got := (&Tmpfile{
				method: tc.method,
				dst:    tc.dst,
				src:    tc.src,
			}).String(); got != tc.want {
				t.Errorf("String() = %v, want %v", got, tc.want)
			}
		})
	}
}
