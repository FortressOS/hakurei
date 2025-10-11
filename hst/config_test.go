package hst_test

import (
	"reflect"
	"testing"

	"hakurei.app/container/fhs"
	"hakurei.app/hst"
)

func TestConfigValidate(t *testing.T) {
	testCases := []struct {
		name    string
		config  *hst.Config
		wantErr error
	}{
		{"nil", nil, &hst.AppError{Step: "validate configuration", Err: hst.ErrConfigNull,
			Msg: "invalid configuration"}},
		{"identity lower", &hst.Config{Identity: -1}, &hst.AppError{Step: "validate configuration", Err: hst.ErrIdentityBounds,
			Msg: "identity -1 out of range"}},
		{"identity upper", &hst.Config{Identity: 10000}, &hst.AppError{Step: "validate configuration", Err: hst.ErrIdentityBounds,
			Msg: "identity 10000 out of range"}},
		{"dbus session", &hst.Config{SessionBus: &hst.BusConfig{See: []string{""}}},
			&hst.BadInterfaceError{Interface: "", Segment: "session"}},
		{"dbus system", &hst.Config{SystemBus: &hst.BusConfig{See: []string{""}}},
			&hst.BadInterfaceError{Interface: "", Segment: "system"}},
		{"container", &hst.Config{}, &hst.AppError{Step: "validate configuration", Err: hst.ErrConfigNull,
			Msg: "configuration missing container state"}},
		{"home", &hst.Config{Container: &hst.ContainerConfig{}}, &hst.AppError{Step: "validate configuration", Err: hst.ErrConfigNull,
			Msg: "container configuration missing path to home directory"}},
		{"shell", &hst.Config{Container: &hst.ContainerConfig{
			Home: fhs.AbsTmp,
		}}, &hst.AppError{Step: "validate configuration", Err: hst.ErrConfigNull,
			Msg: "container configuration missing path to shell"}},
		{"path", &hst.Config{Container: &hst.ContainerConfig{
			Home:  fhs.AbsTmp,
			Shell: fhs.AbsTmp,
		}}, &hst.AppError{Step: "validate configuration", Err: hst.ErrConfigNull,
			Msg: "container configuration missing path to initial program"}},
		{"valid", &hst.Config{Container: &hst.ContainerConfig{
			Home:  fhs.AbsTmp,
			Shell: fhs.AbsTmp,
			Path:  fhs.AbsTmp,
		}}, nil},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.config.Validate(); !reflect.DeepEqual(err, tc.wantErr) {
				t.Errorf("Validate: error = %#v, want %#v", err, tc.wantErr)
			}
		})
	}
}

func TestExtraPermConfig(t *testing.T) {
	testCases := []struct {
		name   string
		config *hst.ExtraPermConfig
		want   string
	}{
		{"nil", nil, "<invalid>"},
		{"nil path", &hst.ExtraPermConfig{Path: nil}, "<invalid>"},
		{"r", &hst.ExtraPermConfig{Path: fhs.AbsRoot, Read: true}, "r--:/"},
		{"r+", &hst.ExtraPermConfig{Ensure: true, Path: fhs.AbsRoot, Read: true}, "r--+:/"},
		{"w", &hst.ExtraPermConfig{Path: hst.AbsPrivateTmp, Write: true}, "-w-:/.hakurei"},
		{"w+", &hst.ExtraPermConfig{Ensure: true, Path: hst.AbsPrivateTmp, Write: true}, "-w-+:/.hakurei"},
		{"x", &hst.ExtraPermConfig{Path: fhs.AbsRunUser, Execute: true}, "--x:/run/user/"},
		{"x+", &hst.ExtraPermConfig{Ensure: true, Path: fhs.AbsRunUser, Execute: true}, "--x+:/run/user/"},
		{"rwx", &hst.ExtraPermConfig{Path: fhs.AbsTmp, Read: true, Write: true, Execute: true}, "rwx:/tmp/"},
		{"rwx+", &hst.ExtraPermConfig{Ensure: true, Path: fhs.AbsTmp, Read: true, Write: true, Execute: true}, "rwx+:/tmp/"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.config.String(); got != tc.want {
				t.Errorf("String: %q, want %q", got, tc.want)
			}
		})
	}
}
