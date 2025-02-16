package system_test

import (
	"strconv"
	"testing"

	"git.gensokyo.uk/security/fortify/internal/system"
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

func TestI_Equal(t *testing.T) {
	testCases := []struct {
		name string
		sys  *system.I
		v    *system.I
		want bool
	}{
		{
			"simple UID",
			system.New(150),
			system.New(150),
			true,
		},
		{
			"simple UID differ",
			system.New(150),
			system.New(151),
			false,
		},
		{
			"simple UID nil",
			system.New(150),
			nil,
			false,
		},
		{
			"op length mismatch",
			system.New(150).
				ChangeHosts("chronos"),
			system.New(150).
				ChangeHosts("chronos").
				Ensure("/run", 0755),
			false,
		},
		{
			"op value mismatch",
			system.New(150).
				ChangeHosts("chronos").
				Ensure("/run", 0644),
			system.New(150).
				ChangeHosts("chronos").
				Ensure("/run", 0755),
			false,
		},
		{
			"op type mismatch",
			system.New(150).
				ChangeHosts("chronos").
				CopyFile(new([]byte), "/home/ophestra/xdg/config/pulse/cookie", 0, 256),
			system.New(150).
				ChangeHosts("chronos").
				Ensure("/run", 0755),
			false,
		},
		{
			"op equals",
			system.New(150).
				ChangeHosts("chronos").
				Ensure("/run", 0755),
			system.New(150).
				ChangeHosts("chronos").
				Ensure("/run", 0755),
			true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.sys.Equal(tc.v) != tc.want {
				t.Errorf("Equal: got %v; want %v",
					!tc.want, tc.want)
			}
		})
	}
}
