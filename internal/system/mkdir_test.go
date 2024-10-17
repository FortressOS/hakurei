package system

import (
	"os"
	"testing"
)

func TestEnsure(t *testing.T) {
	testCases := []struct {
		name string
		perm os.FileMode
	}{
		{"/tmp/fortify.1971", 0701},
		{"/tmp/fortify.1971/tmpdir", 0700},
		{"/tmp/fortify.1971/tmpdir/150", 0700},
		{"/run/user/1971/fortify", 0700},
	}
	for _, tc := range testCases {
		t.Run(tc.name+"_"+tc.perm.String(), func(t *testing.T) {
			sys := New(150)
			sys.Ensure(tc.name, tc.perm)
			(&tcOp{User, tc.name}).test(t, sys.ops, []Op{&Mkdir{User, tc.name, tc.perm, false}}, "Ensure")
		})
	}
}

func TestEphemeral(t *testing.T) {
	testCases := []struct {
		perm os.FileMode
		tcOp
	}{
		{0700, tcOp{Process, "/run/user/1971/fortify/ec07546a772a07cde87389afc84ffd13"}},
		{0701, tcOp{Process, "/tmp/fortify.1971/ec07546a772a07cde87389afc84ffd13"}},
	}
	for _, tc := range testCases {
		t.Run(tc.path+"_"+tc.perm.String()+"_"+TypeString(tc.et), func(t *testing.T) {
			sys := New(150)
			sys.Ephemeral(tc.et, tc.path, tc.perm)
			tc.test(t, sys.ops, []Op{&Mkdir{tc.et, tc.path, tc.perm, true}}, "Ephemeral")
		})
	}
}

func TestMkdir_String(t *testing.T) {
	testCases := []struct {
		want      string
		ephemeral bool
		et        Enablement
	}{
		{"Ensure", false, User},
		{"Ensure", false, Process},
		{"Ensure", false, EWayland},

		{"Wayland", true, EWayland},
		{"X11", true, EX11},
		{"D-Bus", true, EDBus},
		{"PulseAudio", true, EPulse},
	}
	for _, tc := range testCases {
		t.Run(tc.want, func(t *testing.T) {
			m := &Mkdir{
				et:        tc.et,
				path:      "/nonexistent",
				perm:      0701,
				ephemeral: tc.ephemeral,
			}
			want := "mode: " + os.FileMode(0701).String() + " type: " + tc.want + " path: \"/nonexistent\""
			if got := m.String(); got != want {
				t.Errorf("String() = %v, want %v", got, want)
			}
		})
	}
}
