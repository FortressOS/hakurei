package vfs_test

import (
	"testing"

	"git.gensokyo.uk/security/hakurei/sandbox/vfs"
)

func TestUnmangle(t *testing.T) {
	testCases := []struct {
		want   string
		sample string
	}{
		{`\, `, `\134\054\040`},
		{`(10) source -- maybe empty string`, `(10)\040source\040--\040maybe empty string`},
	}

	for _, tc := range testCases {
		t.Run(tc.want, func(t *testing.T) {
			got := vfs.Unmangle(tc.sample)
			if got != tc.want {
				t.Errorf("Unmangle: %q, want %q",
					got, tc.want)
			}
		})
	}
}
