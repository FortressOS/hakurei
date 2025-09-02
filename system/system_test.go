package system_test

import (
	"reflect"
	"strconv"
	"testing"
	_ "unsafe"

	"hakurei.app/system"
)

//go:linkname criteriaHasType hakurei.app/system.(*Criteria).hasType
func criteriaHasType(_ *system.Criteria, _ system.Enablement) bool

func TestCriteria(t *testing.T) {
	testCases := []struct {
		name  string
		ec, t system.Enablement
		want  bool
	}{
		{"nil", 0xff, system.EWayland, true},
		{"nil user", 0xff, system.User, false},
		{"all", system.EWayland | system.EX11 | system.EDBus | system.EPulse | system.User | system.Process, system.Process, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var criteria *system.Criteria
			if tc.ec != 0xff {
				criteria = (*system.Criteria)(&tc.ec)
			}
			if got := criteriaHasType(criteria, tc.t); got != tc.want {
				t.Errorf("hasType: got %v, want %v",
					got, tc.want)
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
		{system.User, "user"},
		{system.Process, "process"},
		{system.User | system.Process, "user, process"},
		{system.EWayland | system.User | system.Process, "wayland, user, process"},
		{system.EX11 | system.Process, "x11, process"},
	}

	for _, tc := range testCases {
		t.Run("label type string "+strconv.Itoa(int(tc.e)), func(t *testing.T) {
			if got := system.TypeString(tc.e); got != tc.want {
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
			system.New(nil, 0)
		})

		t.Run("uid", func(t *testing.T) {
			defer func() {
				want := "invalid call to New"
				if r := recover(); r != want {
					t.Errorf("recover: %v, want %v", r, want)
				}
			}()
			system.New(t.Context(), -1)
		})
	})

	sys := system.New(t.Context(), 0xdeadbeef)
	if got := reflect.ValueOf(sys).Elem().FieldByName("ctx"); got.IsNil() {
		t.Errorf("New: ctx = %#v", got)
	}
	if got := sys.UID(); got != 0xdeadbeef {
		t.Errorf("UID: %d", got)
	}
}

func TestEqual(t *testing.T) {
	testCases := []struct {
		name string
		sys  *system.I
		v    *system.I
		want bool
	}{
		{"simple UID",
			system.New(t.Context(), 150),
			system.New(t.Context(), 150),
			true},

		{"simple UID differ",
			system.New(t.Context(), 150),
			system.New(t.Context(), 151),
			false},

		{"simple UID nil",
			system.New(t.Context(), 150),
			nil,
			false},

		{"op length mismatch",
			system.New(t.Context(), 150).
				ChangeHosts("chronos"),
			system.New(t.Context(), 150).
				ChangeHosts("chronos").
				Ensure("/run", 0755),
			false},

		{"op value mismatch",
			system.New(t.Context(), 150).
				ChangeHosts("chronos").
				Ensure("/run", 0644),
			system.New(t.Context(), 150).
				ChangeHosts("chronos").
				Ensure("/run", 0755),
			false},

		{"op type mismatch",
			system.New(t.Context(), 150).
				ChangeHosts("chronos").
				CopyFile(new([]byte), "/home/ophestra/xdg/config/pulse/cookie", 0, 256),
			system.New(t.Context(), 150).
				ChangeHosts("chronos").
				Ensure("/run", 0755),
			false},

		{"op equals",
			system.New(t.Context(), 150).
				ChangeHosts("chronos").
				Ensure("/run", 0755),
			system.New(t.Context(), 150).
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
