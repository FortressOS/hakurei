package container

import (
	"os"
	"testing"
)

func TestParentPerm(t *testing.T) {
	testCases := []struct {
		perm os.FileMode
		want os.FileMode
	}{
		{0755, 0755},
		{0750, 0750},
		{0705, 0705},
		{0700, 0700},
		{050, 0750},
		{05, 0705},
		{0, 0700},
	}

	for _, tc := range testCases {
		t.Run(tc.perm.String(), func(t *testing.T) {
			if got := parentPerm(tc.perm); got != tc.want {
				t.Errorf("parentPerm: %#o, want %#o", got, tc.want)
			}
		})
	}
}

func TestEscapeOverlayDataSegment(t *testing.T) {
	testCases := []struct {
		name string
		s    string
		want string
	}{
		{"zero", zeroString, zeroString},
		{"multi", `\\\:,:,\\\`, `\\\\\\\:\,\:\,\\\\\\`},
		{"bwrap", `/path :,\`, `/path \:\,\\`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := EscapeOverlayDataSegment(tc.s); got != tc.want {
				t.Errorf("escapeOverlayDataSegment: %s, want %s", got, tc.want)
			}
		})
	}
}
