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
			sys := New(t.Context(), 150)
			sys.CopyFile(new([]byte), tc.path, tc.cap, tc.n)
			tc.test(t, sys.ops, []Op{
				&tmpfileOp{nil, tc.path, tc.n, nil},
			}, "CopyFile")
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
			if got := (&tmpfileOp{src: tc.src, n: tc.n}).String(); got != tc.want {
				t.Errorf("String() = %v, want %v", got, tc.want)
			}
		})
	}
}
