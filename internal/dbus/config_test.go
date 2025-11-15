package dbus_test

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"hakurei.app/hst"
	"hakurei.app/internal/dbus"
)

func TestConfigArgs(t *testing.T) {
	t.Parallel()

	for _, tc := range testCasesExt {
		if tc.wantErr {
			// args does not check for nulls
			continue
		}

		t.Run("build arguments for "+tc.id, func(t *testing.T) {
			t.Parallel()

			if got := dbus.Args(tc.c, tc.bus); !slices.Equal(got, tc.want) {
				t.Errorf("Args: %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNewConfig(t *testing.T) {
	t.Parallel()
	ids := [...]string{"org.chromium.Chromium", "dev.vencord.Vesktop"}

	type newTestCase struct {
		id   string
		args [2]bool
		want *hst.BusConfig
	}

	// populate tests from IDs in generic tests
	tcs := make([]newTestCase, 0, (len(ids)+1)*4)
	// tests for defaults without id
	tcs = append(tcs,
		newTestCase{"", [2]bool{false, false}, &hst.BusConfig{
			Call:      make(map[string]string),
			Broadcast: make(map[string]string),
			Filter:    true,
		}},
		newTestCase{"", [2]bool{false, true}, &hst.BusConfig{
			Call:      make(map[string]string),
			Broadcast: make(map[string]string),
			Filter:    true,
		}},
		newTestCase{"", [2]bool{true, false}, &hst.BusConfig{
			Talk:      []string{"org.freedesktop.DBus", "org.freedesktop.Notifications"},
			Call:      map[string]string{"org.freedesktop.portal.*": "*"},
			Broadcast: map[string]string{"org.freedesktop.portal.*": "@/org/freedesktop/portal/*"},
			Filter:    true,
		}},
		newTestCase{"", [2]bool{true, true}, &hst.BusConfig{
			Talk:      []string{"org.freedesktop.DBus", "org.freedesktop.Notifications"},
			Call:      map[string]string{"org.freedesktop.portal.*": "*"},
			Broadcast: map[string]string{"org.freedesktop.portal.*": "@/org/freedesktop/portal/*"},
			Filter:    true,
		}},
	)
	for _, id := range ids {
		tcs = append(tcs,
			newTestCase{id, [2]bool{false, false}, &hst.BusConfig{
				Call:      make(map[string]string),
				Broadcast: make(map[string]string),
				Filter:    true,
			}},
			newTestCase{id, [2]bool{false, true}, &hst.BusConfig{
				Call:      make(map[string]string),
				Broadcast: make(map[string]string),
				Filter:    true,
			}},
			newTestCase{id, [2]bool{true, false}, &hst.BusConfig{
				Talk:      []string{"org.freedesktop.DBus", "org.freedesktop.Notifications"},
				Own:       []string{id + ".*"},
				Call:      map[string]string{"org.freedesktop.portal.*": "*"},
				Broadcast: map[string]string{"org.freedesktop.portal.*": "@/org/freedesktop/portal/*"},
				Filter:    true,
			}},
			newTestCase{id, [2]bool{true, true}, &hst.BusConfig{
				Talk:      []string{"org.freedesktop.DBus", "org.freedesktop.Notifications"},
				Own:       []string{id + ".*", "org.mpris.MediaPlayer2." + id + ".*"},
				Call:      map[string]string{"org.freedesktop.portal.*": "*"},
				Broadcast: map[string]string{"org.freedesktop.portal.*": "@/org/freedesktop/portal/*"},
				Filter:    true,
			}},
		)
	}

	for _, tc := range tcs {
		name := new(strings.Builder)
		name.WriteString("create new configuration struct")

		if tc.args[0] {
			name.WriteString(" with builtin defaults")
			if tc.args[1] {
				name.WriteString(" (mpris)")
			}
		}

		if tc.id != "" {
			name.WriteString(" for application ID ")
			name.WriteString(tc.id)
		}

		t.Run(name.String(), func(t *testing.T) {
			t.Parallel()

			if gotC := dbus.NewConfig(tc.id, tc.args[0], tc.args[1]); !reflect.DeepEqual(gotC, tc.want) {
				t.Errorf("NewConfig(%q, %t, %t) = %v, want %v",
					tc.id, tc.args[0], tc.args[1],
					gotC, tc.want)
			}
		})
	}
}
