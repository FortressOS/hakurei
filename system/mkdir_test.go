package system

import (
	"os"
	"testing"

	"hakurei.app/container"
)

func TestEnsure(t *testing.T) {
	testCases := []struct {
		name string
		perm os.FileMode
	}{
		{"/tmp/hakurei.1971", 0701},
		{"/tmp/hakurei.1971/tmpdir", 0700},
		{"/tmp/hakurei.1971/tmpdir/150", 0700},
		{"/run/user/1971/hakurei", 0700},
	}
	for _, tc := range testCases {
		t.Run(tc.name+"_"+tc.perm.String(), func(t *testing.T) {
			sys := New(t.Context(), 150)
			sys.Ensure(tc.name, tc.perm)
			(&tcOp{User, tc.name}).test(t, sys.ops, []Op{&mkdirOp{User, tc.name, tc.perm, false}}, "Ensure")
		})
	}
}

func TestEphemeral(t *testing.T) {
	testCases := []struct {
		perm os.FileMode
		tcOp
	}{
		{0700, tcOp{Process, "/run/user/1971/hakurei/ec07546a772a07cde87389afc84ffd13"}},
		{0701, tcOp{Process, "/tmp/hakurei.1971/ec07546a772a07cde87389afc84ffd13"}},
	}
	for _, tc := range testCases {
		t.Run(tc.path+"_"+tc.perm.String()+"_"+TypeString(tc.et), func(t *testing.T) {
			sys := New(t.Context(), 150)
			sys.Ephemeral(tc.et, tc.path, tc.perm)
			tc.test(t, sys.ops, []Op{&mkdirOp{tc.et, tc.path, tc.perm, true}}, "Ephemeral")
		})
	}
}

func TestMkdirString(t *testing.T) {
	testCases := []struct {
		want      string
		ephemeral bool
		et        Enablement
	}{
		{"ensure", false, User},
		{"ensure", false, Process},
		{"ensure", false, EWayland},

		{"wayland", true, EWayland},
		{"x11", true, EX11},
		{"dbus", true, EDBus},
		{"pulseaudio", true, EPulse},
	}
	for _, tc := range testCases {
		t.Run(tc.want, func(t *testing.T) {
			m := &mkdirOp{
				et:        tc.et,
				path:      container.Nonexistent,
				perm:      0701,
				ephemeral: tc.ephemeral,
			}
			want := "mode: " + os.FileMode(0701).String() + " type: " + tc.want + ` path: "/proc/nonexistent"`
			if got := m.String(); got != want {
				t.Errorf("String() = %v, want %v", got, want)
			}
		})
	}
}
