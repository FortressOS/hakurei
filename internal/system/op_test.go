package system_test

import (
	"strconv"
	"testing"

	"git.ophivana.moe/cat/fortify/internal/system"
)

func TestNew(t *testing.T) {
	testCases := []struct {
		uid int
	}{
		{150},
		{149},
		{148},
		{147},
	}

	for _, tc := range testCases {
		t.Run("sys initialised with uid "+strconv.Itoa(tc.uid), func(t *testing.T) {
			if got := system.New(tc.uid); got.UID() != tc.uid {
				t.Errorf("New(%d) uid = %d, want %d",
					tc.uid,
					got.UID(), tc.uid)
			}
		})
	}
}

func TestTypeString(t *testing.T) {
	testCases := []struct {
		e    system.Enablement
		want string
	}{
		{system.EWayland, system.EWayland.String()},
		{system.EX11, system.EX11.String()},
		{system.EDBus, system.EDBus.String()},
		{system.EPulse, system.EPulse.String()},
		{system.User, "User"},
		{system.Process, "Process"},
	}

	for _, tc := range testCases {
		t.Run("label type string "+tc.want, func(t *testing.T) {
			if got := system.TypeString(tc.e); got != tc.want {
				t.Errorf("TypeString(%d) = %v, want %v",
					tc.e,
					got, tc.want)
			}
		})
	}
}
