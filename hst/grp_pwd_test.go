package hst_test

import (
	"strconv"
	"testing"

	"hakurei.app/hst"
)

func TestUIDString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		val  uint32
		want string
	}{
		{hst.AppStart + hst.IdentityStart, "u0_a0"},                           // uidStart
		{hst.ToUser[uint32](hst.RangeSize-1, hst.IdentityEnd), "u9999_a9999"}, // uidEnd

		{hst.IsolatedStart + hst.IdentityStart, "u0_i0"},                    // isolatedStart
		{(hst.RangeSize-1)*hst.UserOffset + hst.IsolatedEnd, "u9999_i9999"}, // isolatedEnd

		{hst.ToUser[uint32](10, 127), "u10_a127"},
		{hst.ToUser[uint32](11, 127), "u11_a127"},

		{0, "0"}, // out of bounds
	}
	for _, tc := range testCases {
		t.Run(strconv.Itoa(int(tc.val)), func(t *testing.T) {
			t.Parallel()

			if got := hst.UID(tc.val).String(); got != tc.want {
				t.Fatalf("UID.String: %q, want %q", got, tc.want)
			}
			if got := hst.GID(tc.val).String(); got != tc.want {
				t.Fatalf("GID.String: %q, want %q", got, tc.want)
			}
		})
	}
}
