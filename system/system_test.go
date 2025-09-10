package system

import (
	"strconv"
	"testing"
)

func TestCriteria(t *testing.T) {
	testCases := []struct {
		name  string
		ec, t Enablement
		want  bool
	}{
		{"nil", 0xff, EWayland, true},
		{"nil user", 0xff, User, false},
		{"all", EWayland | EX11 | EDBus | EPulse | User | Process, Process, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var criteria *Criteria
			if tc.ec != 0xff {
				criteria = (*Criteria)(&tc.ec)
			}
			if got := criteria.hasType(tc.t); got != tc.want {
				t.Errorf("hasType: got %v, want %v",
					got, tc.want)
			}
		})
	}
}

func TestTypeString(t *testing.T) {
	testCases := []struct {
		e    Enablement
		want string
	}{
		{EWayland, EWayland.String()},
		{EX11, EX11.String()},
		{EDBus, EDBus.String()},
		{EPulse, EPulse.String()},
		{User, "user"},
		{Process, "process"},
		{User | Process, "user, process"},
		{EWayland | User | Process, "wayland, user, process"},
		{EX11 | Process, "x11, process"},
	}

	for _, tc := range testCases {
		t.Run("label type string "+strconv.Itoa(int(tc.e)), func(t *testing.T) {
			if got := TypeString(tc.e); got != tc.want {
				t.Errorf("TypeString: %q, want %q", got, tc.want)
			}
		})
	}
}

func TestNew(t *testing.T) {
	t.Run("panic", func(t *testing.T) {
		t.Run("ctx", func(t *testing.T) {
			defer func() {
				want := "invalid call to New"
				if r := recover(); r != want {
					t.Errorf("recover: %v, want %v", r, want)
				}
			}()
			New(nil, 0)
		})

		t.Run("uid", func(t *testing.T) {
			defer func() {
				want := "invalid call to New"
				if r := recover(); r != want {
					t.Errorf("recover: %v, want %v", r, want)
				}
			}()
			New(t.Context(), -1)
		})
	})

	sys := New(t.Context(), 0xdeadbeef)
	if sys.ctx == nil {
		t.Error("New: ctx = nil")
	}
	if got := sys.UID(); got != 0xdeadbeef {
		t.Errorf("UID: %d", got)
	}
}

func TestEqual(t *testing.T) {
	testCases := []struct {
		name string
		sys  *I
		v    *I
		want bool
	}{
		{"simple UID",
			New(t.Context(), 150),
			New(t.Context(), 150),
			true},

		{"simple UID differ",
			New(t.Context(), 150),
			New(t.Context(), 151),
			false},

		{"simple UID nil",
			New(t.Context(), 150),
			nil,
			false},

		{"op length mismatch",
			New(t.Context(), 150).
				ChangeHosts("chronos"),
			New(t.Context(), 150).
				ChangeHosts("chronos").
				Ensure("/run", 0755),
			false},

		{"op value mismatch",
			New(t.Context(), 150).
				ChangeHosts("chronos").
				Ensure("/run", 0644),
			New(t.Context(), 150).
				ChangeHosts("chronos").
				Ensure("/run", 0755),
			false},

		{"op type mismatch",
			New(t.Context(), 150).
				ChangeHosts("chronos").
				CopyFile(new([]byte), "/home/ophestra/xdg/config/pulse/cookie", 0, 256),
			New(t.Context(), 150).
				ChangeHosts("chronos").
				Ensure("/run", 0755),
			false},

		{"op equals",
			New(t.Context(), 150).
				ChangeHosts("chronos").
				Ensure("/run", 0755),
			New(t.Context(), 150).
				ChangeHosts("chronos").
				Ensure("/run", 0755),
			true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.sys.Equal(tc.v) != tc.want {
				t.Errorf("Equal: %v, want %v", !tc.want, tc.want)
			}
		})
	}
}
