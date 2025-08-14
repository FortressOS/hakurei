package hst_test

import (
	"encoding/json"
	"errors"
	"syscall"
	"testing"

	"hakurei.app/hst"
	"hakurei.app/system"
)

func TestEnablements(t *testing.T) {
	testCases := []struct {
		name  string
		e     *hst.Enablements
		data  string
		sData string
	}{
		{"nil", nil, "null", `{"value":null,"magic":3236757504}`},
		{"zero", hst.NewEnablements(0), `{}`, `{"value":{},"magic":3236757504}`},
		{"wayland", hst.NewEnablements(system.EWayland), `{"wayland":true}`, `{"value":{"wayland":true},"magic":3236757504}`},
		{"x11", hst.NewEnablements(system.EX11), `{"x11":true}`, `{"value":{"x11":true},"magic":3236757504}`},
		{"dbus", hst.NewEnablements(system.EDBus), `{"dbus":true}`, `{"value":{"dbus":true},"magic":3236757504}`},
		{"pulse", hst.NewEnablements(system.EPulse), `{"pulse":true}`, `{"value":{"pulse":true},"magic":3236757504}`},
		{"all", hst.NewEnablements(system.EWayland | system.EX11 | system.EDBus | system.EPulse), `{"wayland":true,"x11":true,"dbus":true,"pulse":true}`, `{"value":{"wayland":true,"x11":true,"dbus":true,"pulse":true},"magic":3236757504}`},
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
			if got := hst.NewEnablements(system.EWayland | system.EPulse).Unwrap(); got != system.EWayland|system.EPulse {
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
