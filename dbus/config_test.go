package dbus_test

import (
	"errors"
	"os"
	"path"
	"reflect"
	"slices"
	"strings"
	"testing"

	"git.gensokyo.uk/security/hakurei/dbus"
)

func TestConfig_Args(t *testing.T) {
	for _, tc := range makeTestCases() {
		if tc.wantErr {
			// args does not check for nulls
			continue
		}

		t.Run("build arguments for "+tc.id, func(t *testing.T) {
			if got := tc.c.Args(tc.bus); !slices.Equal(got, tc.want) {
				t.Errorf("Args(%q) = %v, want %v",
					tc.bus,
					got, tc.want)
			}
		})
	}
}

func TestNewConfigFromFile(t *testing.T) {
	for _, tc := range makeTestCases() {
		name := new(strings.Builder)
		name.WriteString("parse configuration file for application ")
		name.WriteString(tc.id)
		if tc.wantErr {
			name.WriteString(" with unexpected results")
		}

		samplePath := path.Join("testdata", tc.id+".json")

		t.Run(name.String(), func(t *testing.T) {
			got, err := dbus.NewConfigFromFile(samplePath)
			if errors.Is(err, os.ErrNotExist) != tc.wantErrF {
				t.Errorf("NewConfigFromFile(%q) error = %v, wantErrF %v",
					samplePath,
					err, tc.wantErrF)
				return
			}

			if tc.wantErrF {
				return
			}

			if !tc.wantErr && !reflect.DeepEqual(got, tc.c) {
				t.Errorf("NewConfigFromFile(%q) got = %v, want %v",
					samplePath,
					got, tc.c)
			}
			if tc.wantErr && reflect.DeepEqual(got, tc.c) {
				t.Errorf("NewConfigFromFile(%q) got = %v, wantErr %v",
					samplePath,
					got, tc.wantErr)
			}
		})
	}
}

func TestNewConfig(t *testing.T) {
	ids := [...]string{"org.chromium.Chromium", "dev.vencord.Vesktop"}

	type newTestCase struct {
		id   string
		args [2]bool
		want *dbus.Config
	}

	// populate tests from IDs in generic tests
	tcs := make([]newTestCase, 0, (len(ids)+1)*4)
	// tests for defaults without id
	tcs = append(tcs,
		newTestCase{"", [2]bool{false, false}, &dbus.Config{
			Call:      make(map[string]string),
			Broadcast: make(map[string]string),
			Filter:    true,
		}},
		newTestCase{"", [2]bool{false, true}, &dbus.Config{
			Call:      make(map[string]string),
			Broadcast: make(map[string]string),
			Filter:    true,
		}},
		newTestCase{"", [2]bool{true, false}, &dbus.Config{
			Talk:      []string{"org.freedesktop.DBus", "org.freedesktop.Notifications"},
			Call:      map[string]string{"org.freedesktop.portal.*": "*"},
			Broadcast: map[string]string{"org.freedesktop.portal.*": "@/org/freedesktop/portal/*"},
			Filter:    true,
		}},
		newTestCase{"", [2]bool{true, true}, &dbus.Config{
			Talk:      []string{"org.freedesktop.DBus", "org.freedesktop.Notifications"},
			Call:      map[string]string{"org.freedesktop.portal.*": "*"},
			Broadcast: map[string]string{"org.freedesktop.portal.*": "@/org/freedesktop/portal/*"},
			Filter:    true,
		}},
	)
	for _, id := range ids {
		tcs = append(tcs,
			newTestCase{id, [2]bool{false, false}, &dbus.Config{
				Call:      make(map[string]string),
				Broadcast: make(map[string]string),
				Filter:    true,
			}},
			newTestCase{id, [2]bool{false, true}, &dbus.Config{
				Call:      make(map[string]string),
				Broadcast: make(map[string]string),
				Filter:    true,
			}},
			newTestCase{id, [2]bool{true, false}, &dbus.Config{
				Talk:      []string{"org.freedesktop.DBus", "org.freedesktop.Notifications"},
				Own:       []string{id + ".*"},
				Call:      map[string]string{"org.freedesktop.portal.*": "*"},
				Broadcast: map[string]string{"org.freedesktop.portal.*": "@/org/freedesktop/portal/*"},
				Filter:    true,
			}},
			newTestCase{id, [2]bool{true, true}, &dbus.Config{
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
			if gotC := dbus.NewConfig(tc.id, tc.args[0], tc.args[1]); !reflect.DeepEqual(gotC, tc.want) {
				t.Errorf("NewConfig(%q, %t, %t) = %v, want %v",
					tc.id, tc.args[0], tc.args[1],
					gotC, tc.want)
			}
		})
	}
}
