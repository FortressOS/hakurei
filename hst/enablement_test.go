package hst_test

import (
	"encoding/json"
	"errors"
	"syscall"
	"testing"

	"hakurei.app/hst"
)

func TestEnablementString(t *testing.T) {
	testCases := []struct {
		flags hst.Enablement
		want  string
	}{
		{0, "(no enablements)"},
		{hst.EWayland, "wayland"},
		{hst.EX11, "x11"},
		{hst.EDBus, "dbus"},
		{hst.EPulse, "pulseaudio"},
		{hst.EWayland | hst.EX11, "wayland, x11"},
		{hst.EWayland | hst.EDBus, "wayland, dbus"},
		{hst.EWayland | hst.EPulse, "wayland, pulseaudio"},
		{hst.EX11 | hst.EDBus, "x11, dbus"},
		{hst.EX11 | hst.EPulse, "x11, pulseaudio"},
		{hst.EDBus | hst.EPulse, "dbus, pulseaudio"},
		{hst.EWayland | hst.EX11 | hst.EDBus, "wayland, x11, dbus"},
		{hst.EWayland | hst.EX11 | hst.EPulse, "wayland, x11, pulseaudio"},
		{hst.EWayland | hst.EDBus | hst.EPulse, "wayland, dbus, pulseaudio"},
		{hst.EX11 | hst.EDBus | hst.EPulse, "x11, dbus, pulseaudio"},
		{hst.EWayland | hst.EX11 | hst.EDBus | hst.EPulse, "wayland, x11, dbus, pulseaudio"},

		{1 << 5, "e20"},
		{1 << 6, "e40"},
		{1 << 7, "e80"},
	}

	for _, tc := range testCases {
		t.Run(tc.want, func(t *testing.T) {
			if got := tc.flags.String(); got != tc.want {
				t.Errorf("String: %q, want %q", got, tc.want)
			}
		})
	}
}

func TestEnablements(t *testing.T) {
	testCases := []struct {
		name  string
		e     *hst.Enablements
		data  string
		sData string
	}{
		{"nil", nil, "null", `{"value":null,"magic":3236757504}`},
		{"zero", hst.NewEnablements(0), `{}`, `{"value":{},"magic":3236757504}`},
		{"wayland", hst.NewEnablements(hst.EWayland), `{"wayland":true}`, `{"value":{"wayland":true},"magic":3236757504}`},
		{"x11", hst.NewEnablements(hst.EX11), `{"x11":true}`, `{"value":{"x11":true},"magic":3236757504}`},
		{"dbus", hst.NewEnablements(hst.EDBus), `{"dbus":true}`, `{"value":{"dbus":true},"magic":3236757504}`},
		{"pulse", hst.NewEnablements(hst.EPulse), `{"pulse":true}`, `{"value":{"pulse":true},"magic":3236757504}`},
		{"all", hst.NewEnablements(hst.EWayland | hst.EX11 | hst.EDBus | hst.EPulse), `{"wayland":true,"x11":true,"dbus":true,"pulse":true}`, `{"value":{"wayland":true,"x11":true,"dbus":true,"pulse":true},"magic":3236757504}`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("marshal", func(t *testing.T) {
				if got, err := json.Marshal(tc.e); err != nil {
					t.Fatalf("Marshal: error = %v", err)
				} else if string(got) != tc.data {
					t.Errorf("Marshal:\n%s, want\n%s", string(got), tc.data)
				}

				if got, err := json.Marshal(struct {
					Value *hst.Enablements `json:"value"`
					Magic int              `json:"magic"`
				}{tc.e, syscall.MS_MGC_VAL}); err != nil {
					t.Fatalf("Marshal: error = %v", err)
				} else if string(got) != tc.sData {
					t.Errorf("Marshal:\n%s, want\n%s", string(got), tc.sData)
				}
			})

			t.Run("unmarshal", func(t *testing.T) {
				{
					got := new(hst.Enablements)
					if err := json.Unmarshal([]byte(tc.data), &got); err != nil {
						t.Fatalf("Unmarshal: error = %v", err)
					}
					if tc.e == nil {
						if got != nil {
							t.Errorf("Unmarshal: %v", got)
						}
					} else if *got != *tc.e {
						t.Errorf("Unmarshal: %v, want %v", got, tc.e)
					}
				}

				{
					got := *(new(struct {
						Value *hst.Enablements `json:"value"`
						Magic int              `json:"magic"`
					}))
					if err := json.Unmarshal([]byte(tc.sData), &got); err != nil {
						t.Fatalf("Unmarshal: error = %v", err)
					}
					if tc.e == nil {
						if got.Value != nil {
							t.Errorf("Unmarshal: %v", got)
						}
					} else if *got.Value != *tc.e {
						t.Errorf("Unmarshal: %v, want %v", got.Value, tc.e)
					}
				}
			})
		})
	}

	t.Run("unwrap", func(t *testing.T) {
		t.Run("nil", func(t *testing.T) {
			if got := (*hst.Enablements)(nil).Unwrap(); got != 0 {
				t.Errorf("Unwrap: %v", got)
			}
		})

		t.Run("val", func(t *testing.T) {
			if got := hst.NewEnablements(hst.EWayland | hst.EPulse).Unwrap(); got != hst.EWayland|hst.EPulse {
				t.Errorf("Unwrap: %v", got)
			}
		})
	})

	t.Run("passthrough", func(t *testing.T) {
		if _, err := (*hst.Enablements)(nil).MarshalJSON(); !errors.Is(err, syscall.EINVAL) {
			t.Errorf("MarshalJSON: error = %v", err)
		}
		if err := (*hst.Enablements)(nil).UnmarshalJSON(nil); !errors.Is(err, syscall.EINVAL) {
			t.Errorf("UnmarshalJSON: error = %v", err)
		}
		if err := new(hst.Enablements).UnmarshalJSON([]byte{}); err == nil {
			t.Errorf("UnmarshalJSON: error = %v", err)
		}
	})
}
