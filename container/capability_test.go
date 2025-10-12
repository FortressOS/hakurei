package container

import "testing"

func TestCapToIndex(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		cap  uintptr
		want uintptr
	}{
		{"CAP_SYS_ADMIN", CAP_SYS_ADMIN, 0},
		{"CAP_SETPCAP", CAP_SETPCAP, 0},
		{"CAP_DAC_OVERRIDE", CAP_DAC_OVERRIDE, 0},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := capToIndex(tc.cap); got != tc.want {
				t.Errorf("capToIndex: %#x, want %#x", got, tc.want)
			}
		})
	}
}

func TestCapToMask(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		cap  uintptr
		want uint32
	}{
		{"CAP_SYS_ADMIN", CAP_SYS_ADMIN, 0x200000},
		{"CAP_SETPCAP", CAP_SETPCAP, 0x100},
		{"CAP_DAC_OVERRIDE", CAP_DAC_OVERRIDE, 0x2},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := capToMask(tc.cap); got != tc.want {
				t.Errorf("capToMask: %#x, want %#x", got, tc.want)
			}
		})
	}
}
