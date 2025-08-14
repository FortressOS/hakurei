package container

import "testing"

func TestToSysroot(t *testing.T) {
	testCases := []struct {
		name string
		want string
	}{
		{"", "/sysroot"},
		{"/", "/sysroot"},
		{"//etc///", "/sysroot/etc"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := toSysroot(tc.name); got != tc.want {
				t.Errorf("toSysroot: %q, want %q", got, tc.want)
			}
		})
	}
}

func TestToHost(t *testing.T) {
	testCases := []struct {
		name string
		want string
	}{
		{"", "/host"},
		{"/", "/host"},
		{"//etc///", "/host/etc"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := toHost(tc.name); got != tc.want {
				t.Errorf("toHost: %q, want %q", got, tc.want)
			}
		})
	}
}

// InternalToHostOvlEscape exports toHost passed to EscapeOverlayDataSegment.
func InternalToHostOvlEscape(s string) string { return EscapeOverlayDataSegment(toHost(s)) }
