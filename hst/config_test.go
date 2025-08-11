package hst_test

import (
	"testing"

	"hakurei.app/container"
	"hakurei.app/hst"
)

func TestExtraPermConfig(t *testing.T) {
	testCases := []struct {
		name   string
		config *hst.ExtraPermConfig
		want   string
	}{
		{"nil", nil, "<invalid>"},
		{"nil path", &hst.ExtraPermConfig{Path: nil}, "<invalid>"},
		{"r", &hst.ExtraPermConfig{Path: container.AbsFHSRoot, Read: true}, "r--:/"},
		{"r+", &hst.ExtraPermConfig{Ensure: true, Path: container.AbsFHSRoot, Read: true}, "r--+:/"},
		{"w", &hst.ExtraPermConfig{Path: hst.AbsTmp, Write: true}, "-w-:/.hakurei"},
		{"w+", &hst.ExtraPermConfig{Ensure: true, Path: hst.AbsTmp, Write: true}, "-w-+:/.hakurei"},
		{"x", &hst.ExtraPermConfig{Path: container.AbsFHSRunUser, Execute: true}, "--x:/run/user/"},
		{"x+", &hst.ExtraPermConfig{Ensure: true, Path: container.AbsFHSRunUser, Execute: true}, "--x+:/run/user/"},
		{"rwx", &hst.ExtraPermConfig{Path: container.AbsFHSTmp, Read: true, Write: true, Execute: true}, "rwx:/tmp/"},
		{"rwx+", &hst.ExtraPermConfig{Ensure: true, Path: container.AbsFHSTmp, Read: true, Write: true, Execute: true}, "rwx+:/tmp/"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.config.String(); got != tc.want {
				t.Errorf("String: %q, want %q", got, tc.want)
			}
		})
	}
}
