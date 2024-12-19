package system

import (
	"strconv"
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
				&Tmpfile{Process, tmpfileLink, tc.dst, tc.src},
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
				&Tmpfile{tc.et, tmpfileLink, tc.dst, tc.path},
			}, "LinkFileType")
		})
	}
}

func TestWrite(t *testing.T) {
	testCases := []struct {
		dst, src string
	}{
		{"/etc/passwd", "chronos:x:65534:65534:Fortify:/home/chronos:/run/current-system/sw/bin/zsh\n"},
		{"/etc/group", "fortify:x:65534:\n"},
	}
	for _, tc := range testCases {
		t.Run("write "+strconv.Itoa(len(tc.src))+" bytes to "+tc.dst, func(t *testing.T) {
			sys := New(150)
			sys.Write(tc.dst, tc.src)
			(&tcOp{Process, "(" + strconv.Itoa(len(tc.src)) + " bytes of data)"}).test(t, sys.ops, []Op{
				&Tmpfile{Process, tmpfileWrite, tc.dst, tc.src},
				&ACL{Process, tc.dst, []acl.Perm{acl.Read}},
			}, "Write")
		})
	}
}

func TestWriteType(t *testing.T) {
	testCases := []struct {
		et       Enablement
		dst, src string
	}{
		{Process, "/etc/passwd", "chronos:x:65534:65534:Fortify:/home/chronos:/run/current-system/sw/bin/zsh\n"},
		{Process, "/etc/group", "fortify:x:65534:\n"},
		{User, "/etc/passwd", "chronos:x:65534:65534:Fortify:/home/chronos:/run/current-system/sw/bin/zsh\n"},
		{User, "/etc/group", "fortify:x:65534:\n"},
	}
	for _, tc := range testCases {
		t.Run("write "+strconv.Itoa(len(tc.src))+" bytes to "+tc.dst+" with type "+TypeString(tc.et), func(t *testing.T) {
			sys := New(150)
			sys.WriteType(tc.et, tc.dst, tc.src)
			(&tcOp{tc.et, "(" + strconv.Itoa(len(tc.src)) + " bytes of data)"}).test(t, sys.ops, []Op{
				&Tmpfile{tc.et, tmpfileWrite, tc.dst, tc.src},
				&ACL{tc.et, tc.dst, []acl.Perm{acl.Read}},
			}, "WriteType")
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
		{tmpfileLink, "/run/user/1971/fortify/4b6bdc9182fb2f1d3a965c5fa8b9b66e/wayland", "/run/user/1971/wayland-0",
			`"/run/user/1971/fortify/4b6bdc9182fb2f1d3a965c5fa8b9b66e/wayland" from "/run/user/1971/wayland-0"`},
		{tmpfileLink, "/run/user/1971/fortify/4b6bdc9182fb2f1d3a965c5fa8b9b66e/pulse", "/run/user/1971/pulse/native",
			`"/run/user/1971/fortify/4b6bdc9182fb2f1d3a965c5fa8b9b66e/pulse" from "/run/user/1971/pulse/native"`},
		{tmpfileWrite, "/tmp/fortify.1971/4b6bdc9182fb2f1d3a965c5fa8b9b66e/passwd", "chronos:x:65534:65534:Fortify:/home/chronos:/run/current-system/sw/bin/zsh\n",
			`75 bytes of data to "/tmp/fortify.1971/4b6bdc9182fb2f1d3a965c5fa8b9b66e/passwd"`},
		{tmpfileWrite, "/tmp/fortify.1971/4b6bdc9182fb2f1d3a965c5fa8b9b66e/group", "fortify:x:65534:\n",
			`17 bytes of data to "/tmp/fortify.1971/4b6bdc9182fb2f1d3a965c5fa8b9b66e/group"`},
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
