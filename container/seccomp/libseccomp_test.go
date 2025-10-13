package seccomp_test

import (
	"crypto/sha512"
	"errors"
	"syscall"
	"testing"

	. "hakurei.app/container/bits"
	. "hakurei.app/container/seccomp"
)

func TestLibraryError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		sample  *LibraryError
		want    string
		wantIs  bool
		compare error
	}{
		{
			"full",
			&LibraryError{Prefix: "seccomp_export_bpf failed", Seccomp: syscall.ECANCELED, Errno: syscall.EBADF},
			"seccomp_export_bpf failed: operation canceled (bad file descriptor)",
			true,
			&LibraryError{Prefix: "seccomp_export_bpf failed", Seccomp: syscall.ECANCELED, Errno: syscall.EBADF},
		},
		{
			"errno only",
			&LibraryError{Prefix: "seccomp_init failed", Errno: syscall.ENOMEM},
			"seccomp_init failed: cannot allocate memory",
			false,
			nil,
		},
		{
			"seccomp only",
			&LibraryError{Prefix: "internal libseccomp failure", Seccomp: syscall.EFAULT},
			"internal libseccomp failure: bad address",
			true,
			syscall.EFAULT,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if errors.Is(tc.sample, tc.compare) != tc.wantIs {
				t.Errorf("errors.Is(%#v, %#v) did not return %v",
					tc.sample, tc.compare, tc.wantIs)
			}

			if got := tc.sample.Error(); got != tc.want {
				t.Errorf("Error: %q, want %q",
					got, tc.want)
			}
		})
	}

	t.Run("invalid", func(t *testing.T) {
		t.Parallel()

		wantPanic := "invalid libseccomp error"
		defer func() {
			if r := recover(); r != wantPanic {
				t.Errorf("panic: %q, want %q", r, wantPanic)
			}
		}()
		_ = new(LibraryError).Error()
	})
}

func TestExport(t *testing.T) {
	t.Parallel()

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

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			want := bpfExpected[bpfPreset{tc.flags, tc.presets}]
			if data, err := Export(Preset(tc.presets, tc.flags), tc.flags); (err != nil) != tc.wantErr {
				t.Errorf("Export: error = %v, wantErr %v", err, tc.wantErr)
				return
			} else if got := sha512.Sum512(data); got != want {
				t.Fatalf("Export: hash = %x, want %x", got, want)
				return
			}
		})
	}
}

func BenchmarkExport(b *testing.B) {
	const exportFlags = AllowMultiarch | AllowCAN | AllowBluetooth
	const presetFlags = PresetExt | PresetDenyNS | PresetDenyTTY | PresetDenyDevel | PresetLinux32
	var want = bpfExpected[bpfPreset{exportFlags, presetFlags}]

	for b.Loop() {
		data, err := Export(Preset(presetFlags, exportFlags), exportFlags)

		b.StopTimer()
		if err != nil {
			b.Fatalf("Export: error = %v", err)
		}
		if got := sha512.Sum512(data); got != want {
			b.Fatalf("Export: hash = %x, want %x", got, want)
			return
		}
		b.StartTimer()
	}
}
