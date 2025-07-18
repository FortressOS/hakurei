package seccomp_test

import (
	"crypto/sha512"
	"errors"
	"io"
	"slices"
	"syscall"
	"testing"

	. "hakurei.app/container/seccomp"
)

func TestExport(t *testing.T) {
	testCases := []struct {
		name    string
		flags   ExportFlag
		presets FilterPreset
		wantErr bool
	}{
		{"everything", AllowMultiarch | AllowCAN |
			AllowBluetooth, PresetExt |
			PresetDenyNS | PresetDenyTTY | PresetDenyDevel |
			PresetLinux32, false},

		{"compat", 0, 0, false},
		{"base", 0, PresetExt, false},
		{"strict", 0, PresetStrict, false},
		{"strict compat", 0, PresetDenyNS | PresetDenyTTY | PresetDenyDevel, false},
		{"hakurei default", 0, PresetExt | PresetDenyDevel, false},
		{"hakurei tty", 0, PresetExt | PresetDenyNS | PresetDenyDevel, false},
	}

	buf := make([]byte, 8)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := New(Preset(tc.presets, tc.flags), tc.flags)
			want := bpfExpected[bpfPreset{tc.flags, tc.presets}]
			digest := sha512.New()

			if _, err := io.CopyBuffer(digest, e, buf); (err != nil) != tc.wantErr {
				t.Errorf("Exporter: error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if err := e.Close(); err != nil {
				t.Errorf("Close: error = %v", err)
			}
			if got := digest.Sum(nil); !slices.Equal(got, want) {
				t.Fatalf("Export() hash = %x, want %x",
					got, want)
				return
			}
		})
	}

	t.Run("close without use", func(t *testing.T) {
		e := New(Preset(0, 0), 0)
		if err := e.Close(); !errors.Is(err, syscall.EINVAL) {
			t.Errorf("Close: error = %v", err)
			return
		}
	})

	t.Run("close partial read", func(t *testing.T) {
		e := New(Preset(0, 0), 0)
		if _, err := e.Read(nil); err != nil {
			t.Errorf("Read: error = %v", err)
			return
		}
		// the underlying implementation uses buffered io, so the outcome of this is nondeterministic;
		// that is not harmful however, so both outcomes are checked for here
		if err := e.Close(); err != nil &&
			(!errors.Is(err, syscall.ECANCELED) || !errors.Is(err, syscall.EBADF)) {
			t.Errorf("Close: error = %v", err)
			return
		}
	})
}

func BenchmarkExport(b *testing.B) {
	buf := make([]byte, 8)
	for b.Loop() {
		e := New(
			Preset(PresetExt|PresetDenyNS|PresetDenyTTY|PresetDenyDevel|PresetLinux32,
				AllowMultiarch|AllowCAN|AllowBluetooth),
			AllowMultiarch|AllowCAN|AllowBluetooth)
		if _, err := io.CopyBuffer(io.Discard, e, buf); err != nil {
			b.Fatalf("cannot export: %v", err)
		}
		if err := e.Close(); err != nil {
			b.Fatalf("cannot close exporter: %v", err)
		}
	}
}
