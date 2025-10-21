package hst_test

import (
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"syscall"
	"testing"

	"hakurei.app/hst"
)

func TestFlagsString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		flags hst.Flags
		want  string
	}{
		{"none", 0, "none"},
		{"none high", hst.FAll + 1, "none"},
		{"all", hst.FAll, "multiarch, compat, devel, userns, net, abstract, tty, mapuid, device, runtime, tmpdir"},
		{"all high", math.MaxUint, "multiarch, compat, devel, userns, net, abstract, tty, mapuid, device, runtime, tmpdir"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := tc.flags.String(); got != tc.want {
				t.Errorf("String(%#b): %q, want %q", tc.flags, got, tc.want)
			}
		})
	}
}

func TestContainerConfig(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		c    *hst.ContainerConfig
		data string
	}{
		{"nil", nil, "null"},
		{"zero", new(hst.ContainerConfig),
			`{"env":null,"filesystem":null,"shell":null,"home":null,"args":null,"map_real_uid":false}`},
		{"seccomp compat", &hst.ContainerConfig{Flags: hst.FSeccompCompat},
			`{"env":null,"filesystem":null,"shell":null,"home":null,"args":null,"seccomp_compat":true,"map_real_uid":false}`},
		{"hostnet hostabstract", &hst.ContainerConfig{Flags: hst.FHostNet | hst.FHostAbstract},
			`{"env":null,"filesystem":null,"shell":null,"home":null,"args":null,"host_net":true,"host_abstract":true,"map_real_uid":false}`},
		{"hostnet hostabstract mapuid", &hst.ContainerConfig{Flags: hst.FHostNet | hst.FHostAbstract | hst.FMapRealUID},
			`{"env":null,"filesystem":null,"shell":null,"home":null,"args":null,"host_net":true,"host_abstract":true,"map_real_uid":true}`},
		{"all", &hst.ContainerConfig{Flags: hst.FAll},
			`{"env":null,"filesystem":null,"shell":null,"home":null,"args":null,"seccomp_compat":true,"devel":true,"userns":true,"host_net":true,"host_abstract":true,"tty":true,"multiarch":true,"map_real_uid":true,"device":true,"share_runtime":true,"share_tmpdir":true}`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			t.Run("marshal", func(t *testing.T) {
				t.Parallel()
				if got, err := json.Marshal(tc.c); err != nil {
					t.Fatalf("Marshal: error = %v", err)
				} else if string(got) != tc.data {
					t.Errorf("Marshal:\n%s, want\n%s", string(got), tc.data)
				}
			})

			t.Run("unmarshal", func(t *testing.T) {
				t.Parallel()

				{
					got := new(hst.ContainerConfig)
					if err := json.Unmarshal([]byte(tc.data), &got); err != nil {
						t.Fatalf("Unmarshal: error = %v", err)
					}
					if !reflect.DeepEqual(got, tc.c) {
						t.Errorf("Unmarshal: %v, want %v", got, tc.c)
					}
				}
			})
		})
	}

	t.Run("passthrough", func(t *testing.T) {
		t.Parallel()

		if _, err := (*hst.ContainerConfig)(nil).MarshalJSON(); !errors.Is(err, syscall.EINVAL) {
			t.Errorf("MarshalJSON: error = %v", err)
		}
		if err := (*hst.ContainerConfig)(nil).UnmarshalJSON(nil); !errors.Is(err, syscall.EINVAL) {
			t.Errorf("UnmarshalJSON: error = %v", err)
		}
		if err := new(hst.ContainerConfig).UnmarshalJSON([]byte{}); err == nil {
			t.Errorf("UnmarshalJSON: error = %v", err)
		}
	})
}
