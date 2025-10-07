package hst_test

import (
	"reflect"
	"slices"
	"testing"

	"hakurei.app/container"
	"hakurei.app/hst"
)

func TestBadInterfaceError(t *testing.T) {
	testCases := []struct {
		name string
		err  error
		want string
	}{
		{"nil", (*hst.BadInterfaceError)(nil), "<nil>"},
		{"session", &hst.BadInterfaceError{Interface: "\x00", Segment: "session"},
			`bad interface string "\x00" in session bus configuration`},
		{"system", &hst.BadInterfaceError{Interface: "\x01", Segment: "system"},
			`bad interface string "\x01" in system bus configuration`},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if gotError := tc.err.Error(); gotError != tc.want {
				t.Errorf("Error: %s, want %s", gotError, tc.want)
			}
			if gotMessage, ok := container.GetErrorMessage(tc.err); !ok {
				t.Error("GetErrorMessage: ok = false")
			} else if gotMessage != tc.want {
				t.Errorf("GetErrorMessage: %s, want %s", gotMessage, tc.want)
			}
		})
	}
}

func TestBusConfigInterfaces(t *testing.T) {
	testCases := []struct {
		name   string
		c      *hst.BusConfig
		cutoff int
		want   []string
	}{
		{"nil", nil, 0, nil},
		{"all", &hst.BusConfig{
			See: []string{"see"}, Talk: []string{"talk"}, Own: []string{"own"},
			Call:      map[string]string{"call": "unreachable"},
			Broadcast: map[string]string{"broadcast": "unreachable"},
		}, 0, []string{"see", "talk", "own", "call", "broadcast"}},

		{"all cutoff", &hst.BusConfig{
			See: []string{"see"}, Talk: []string{"talk"}, Own: []string{"own"},
			Call:      map[string]string{"call": "unreachable"},
			Broadcast: map[string]string{"broadcast": "unreachable"},
		}, 3, []string{"see", "talk", "own"}},

		{"cutoff see", &hst.BusConfig{See: []string{"see"}}, 1, []string{"see"}},
		{"cutoff talk", &hst.BusConfig{Talk: []string{"talk"}}, 1, []string{"talk"}},
		{"cutoff own", &hst.BusConfig{Own: []string{"own"}}, 1, []string{"own"}},
		{"cutoff call", &hst.BusConfig{Call: map[string]string{"call": "unreachable"}}, 1, []string{"call"}},
		{"cutoff broadcast", &hst.BusConfig{Broadcast: map[string]string{"broadcast": "unreachable"}}, 1, []string{"broadcast"}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			if tc.cutoff > 0 {
				var i int
				got = make([]string, 0, tc.cutoff)
				for v := range tc.c.Interfaces {
					i++
					got = append(got, v)
					if i == tc.cutoff {
						break
					}
				}
			} else {
				got = slices.Collect(tc.c.Interfaces)
			}

			if !slices.Equal(got, tc.want) {
				t.Errorf("Interfaces: %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBusConfigCheckInterfaces(t *testing.T) {
	testCases := []struct {
		name string
		c    *hst.BusConfig
		err  error
	}{
		{"nil", nil, nil},
		{"zero", &hst.BusConfig{See: []string{""}},
			&hst.BadInterfaceError{Interface: "", Segment: "zero"}},
		{"suffix", &hst.BusConfig{See: []string{".*"}},
			&hst.BadInterfaceError{Interface: ".*", Segment: "suffix"}},
		{"valid suffix", &hst.BusConfig{See: []string{"..*"}}, nil},
		{"valid", &hst.BusConfig{See: []string{"."}}, nil},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.c.CheckInterfaces(tc.name); !reflect.DeepEqual(err, tc.err) {
				t.Errorf("CheckInterfaces: error = %#v, want %#v", err, tc.err)
			}
		})
	}
}
