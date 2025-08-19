package container

import "testing"

func TestIsAutoRootBindable(t *testing.T) {
	testCases := []struct {
		name string
		want bool
	}{
		{"proc", false},
		{"dev", false},
		{"tmp", false},
		{"mnt", false},
		{"etc", false},
		{"", false},

		{"var", true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsAutoRootBindable(tc.name); got != tc.want {
				t.Errorf("IsAutoRootBindable: %v, want %v", got, tc.want)
			}
		})
	}
}
