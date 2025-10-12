package check_test

import (
	"testing"

	"hakurei.app/container/check"
)

func TestEscapeOverlayDataSegment(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		s    string
		want string
	}{
		{"zero", "", ""},
		{"multi", `\\\:,:,\\\`, `\\\\\\\:\,\:\,\\\\\\`},
		{"bwrap", `/path :,\`, `/path \:\,\\`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := check.EscapeOverlayDataSegment(tc.s); got != tc.want {
				t.Errorf("escapeOverlayDataSegment: %s, want %s", got, tc.want)
			}
		})
	}
}
