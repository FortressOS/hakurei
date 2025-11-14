package ldd_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"hakurei.app/container/check"
	"hakurei.app/ldd"
)

func TestEntryUnexpectedSegmentsError(t *testing.T) {
	const want = `unexpected segments in entry "\x00"`
	if got := ldd.EntryUnexpectedSegmentsError("\x00").Error(); got != want {
		t.Fatalf("Error: %s, want %s", got, want)
	}
}

func TestDecodeError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name, out string
		wantErr   error
	}{
		{"unexpected newline", `
/lib/ld-musl-x86_64.so.1 (0x7ff71c0a4000)

libzstd.so.1 => /usr/lib/libzstd.so.1 (0x7ff71bfd2000)
`, ldd.ErrUnexpectedNewline},

		{"unexpected separator", `
libzstd.so.1 = /usr/lib/libzstd.so.1 (0x7ff71bfd2000)
`, ldd.ErrUnexpectedSeparator},

		{"path not absolute", `
libzstd.so.1 => usr/lib/libzstd.so.1 (0x7ff71bfd2000)
`, &check.AbsoluteError{Pathname: "usr/lib/libzstd.so.1"}},

		{"unexpected segments", `
meow libzstd.so.1 => /usr/lib/libzstd.so.1 (0x7ff71bfd2000)
`, ldd.EntryUnexpectedSegmentsError("meow libzstd.so.1 => /usr/lib/libzstd.so.1 (0x7ff71bfd2000)")},

		{"bad location format", `
libzstd.so.1 => /usr/lib/libzstd.so.1 7ff71bfd2000
`, ldd.ErrBadLocationFormat},

		{"valid", ``, nil},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			d := ldd.NewDecoder(strings.NewReader(tc.out))

			if _, err := d.Decode(); !reflect.DeepEqual(err, tc.wantErr) {
				t.Errorf("Decode: error = %v, wantErr %v", err, tc.wantErr)
			}
			if d.Scan(new(ldd.Entry)) {
				t.Fatalf("Scan: unexpected true")
			}
		})
	}
}

func TestParse(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		file, out string
		want      []*ldd.Entry
		paths     []*check.Absolute
	}{
		{"musl /bin/kmod", `
/lib/ld-musl-x86_64.so.1 (0x7ff71c0a4000)
libzstd.so.1 => /usr/lib/libzstd.so.1 (0x7ff71bfd2000)
liblzma.so.5 => /usr/lib/liblzma.so.5 (0x7ff71bf9a000)
libz.so.1 => /lib/libz.so.1 (0x7ff71bf80000)
libcrypto.so.3 => /lib/libcrypto.so.3 (0x7ff71ba00000)
libc.musl-x86_64.so.1 => /lib/ld-musl-x86_64.so.1 (0x7ff71c0a4000)`, []*ldd.Entry{
			{"/lib/ld-musl-x86_64.so.1", nil, 0x7ff71c0a4000},
			{"libzstd.so.1", check.MustAbs("/usr/lib/libzstd.so.1"), 0x7ff71bfd2000},
			{"liblzma.so.5", check.MustAbs("/usr/lib/liblzma.so.5"), 0x7ff71bf9a000},
			{"libz.so.1", check.MustAbs("/lib/libz.so.1"), 0x7ff71bf80000},
			{"libcrypto.so.3", check.MustAbs("/lib/libcrypto.so.3"), 0x7ff71ba00000},
			{"libc.musl-x86_64.so.1", check.MustAbs("/lib/ld-musl-x86_64.so.1"), 0x7ff71c0a4000},
		}, []*check.Absolute{
			check.MustAbs("/lib"),
			check.MustAbs("/usr/lib"),
		}},

		{"glibc /nix/store/rc3n2r3nffpib2gqpxlkjx36frw6n34z-kmod-31/bin/kmod", `
	linux-vdso.so.1 (0x00007ffed65be000)
	libzstd.so.1 => /nix/store/80pxmvb9q43kh9rkjagc4h41vf6dh1y6-zstd-1.5.6/lib/libzstd.so.1 (0x00007f3199cd1000)
	liblzma.so.5 => /nix/store/g78jna1i5qhh8gqs4mr64648f0szqgw4-xz-5.4.7/lib/liblzma.so.5 (0x00007f3199ca2000)
	libc.so.6 => /nix/store/c10zhkbp6jmyh0xc5kd123ga8yy2p4hk-glibc-2.39-52/lib/libc.so.6 (0x00007f3199ab5000)
	libpthread.so.0 => /nix/store/c10zhkbp6jmyh0xc5kd123ga8yy2p4hk-glibc-2.39-52/lib/libpthread.so.0 (0x00007f3199ab0000)
	/nix/store/c10zhkbp6jmyh0xc5kd123ga8yy2p4hk-glibc-2.39-52/lib/ld-linux-x86-64.so.2 => /nix/store/c10zhkbp6jmyh0xc5kd123ga8yy2p4hk-glibc-2.39-52/lib64/ld-linux-x86-64.so.2 (0x00007f3199da5000)`, []*ldd.Entry{
			{"linux-vdso.so.1", nil, 0x00007ffed65be000},
			{"libzstd.so.1", check.MustAbs("/nix/store/80pxmvb9q43kh9rkjagc4h41vf6dh1y6-zstd-1.5.6/lib/libzstd.so.1"), 0x00007f3199cd1000},
			{"liblzma.so.5", check.MustAbs("/nix/store/g78jna1i5qhh8gqs4mr64648f0szqgw4-xz-5.4.7/lib/liblzma.so.5"), 0x00007f3199ca2000},
			{"libc.so.6", check.MustAbs("/nix/store/c10zhkbp6jmyh0xc5kd123ga8yy2p4hk-glibc-2.39-52/lib/libc.so.6"), 0x00007f3199ab5000},
			{"libpthread.so.0", check.MustAbs("/nix/store/c10zhkbp6jmyh0xc5kd123ga8yy2p4hk-glibc-2.39-52/lib/libpthread.so.0"), 0x00007f3199ab0000},
			{"/nix/store/c10zhkbp6jmyh0xc5kd123ga8yy2p4hk-glibc-2.39-52/lib/ld-linux-x86-64.so.2", check.MustAbs("/nix/store/c10zhkbp6jmyh0xc5kd123ga8yy2p4hk-glibc-2.39-52/lib64/ld-linux-x86-64.so.2"), 0x00007f3199da5000},
		}, []*check.Absolute{
			check.MustAbs("/nix/store/80pxmvb9q43kh9rkjagc4h41vf6dh1y6-zstd-1.5.6/lib"),
			check.MustAbs("/nix/store/c10zhkbp6jmyh0xc5kd123ga8yy2p4hk-glibc-2.39-52/lib"),
			check.MustAbs("/nix/store/c10zhkbp6jmyh0xc5kd123ga8yy2p4hk-glibc-2.39-52/lib64"),
			check.MustAbs("/nix/store/g78jna1i5qhh8gqs4mr64648f0szqgw4-xz-5.4.7/lib"),
		}},

		{"glibc /usr/bin/xdg-dbus-proxy", `
	linux-vdso.so.1 (0x00007725f5772000)
	libglib-2.0.so.0 => /usr/lib/libglib-2.0.so.0 (0x00007725f55d5000)
	libgio-2.0.so.0 => /usr/lib/libgio-2.0.so.0 (0x00007725f5406000)
	libgobject-2.0.so.0 => /usr/lib/libgobject-2.0.so.0 (0x00007725f53a6000)
	libgcc_s.so.1 => /usr/lib/libgcc_s.so.1 (0x00007725f5378000)
	libc.so.6 => /usr/lib/libc.so.6 (0x00007725f5187000)
	libpcre2-8.so.0 => /usr/lib/libpcre2-8.so.0 (0x00007725f50e8000)
	libgmodule-2.0.so.0 => /usr/lib/libgmodule-2.0.so.0 (0x00007725f50df000)
	libz.so.1 => /usr/lib/libz.so.1 (0x00007725f50c6000)
	libmount.so.1 => /usr/lib/libmount.so.1 (0x00007725f5076000)
	libffi.so.8 => /usr/lib/libffi.so.8 (0x00007725f506b000)
	/lib64/ld-linux-x86-64.so.2 => /usr/lib64/ld-linux-x86-64.so.2 (0x00007725f5774000)
	libblkid.so.1 => /usr/lib/libblkid.so.1 (0x00007725f5032000)`, []*ldd.Entry{
			{"linux-vdso.so.1", nil, 0x00007725f5772000},
			{"libglib-2.0.so.0", check.MustAbs("/usr/lib/libglib-2.0.so.0"), 0x00007725f55d5000},
			{"libgio-2.0.so.0", check.MustAbs("/usr/lib/libgio-2.0.so.0"), 0x00007725f5406000},
			{"libgobject-2.0.so.0", check.MustAbs("/usr/lib/libgobject-2.0.so.0"), 0x00007725f53a6000},
			{"libgcc_s.so.1", check.MustAbs("/usr/lib/libgcc_s.so.1"), 0x00007725f5378000},
			{"libc.so.6", check.MustAbs("/usr/lib/libc.so.6"), 0x00007725f5187000},
			{"libpcre2-8.so.0", check.MustAbs("/usr/lib/libpcre2-8.so.0"), 0x00007725f50e8000},
			{"libgmodule-2.0.so.0", check.MustAbs("/usr/lib/libgmodule-2.0.so.0"), 0x00007725f50df000},
			{"libz.so.1", check.MustAbs("/usr/lib/libz.so.1"), 0x00007725f50c6000},
			{"libmount.so.1", check.MustAbs("/usr/lib/libmount.so.1"), 0x00007725f5076000},
			{"libffi.so.8", check.MustAbs("/usr/lib/libffi.so.8"), 0x00007725f506b000},
			{"/lib64/ld-linux-x86-64.so.2", check.MustAbs("/usr/lib64/ld-linux-x86-64.so.2"), 0x00007725f5774000},
			{"libblkid.so.1", check.MustAbs("/usr/lib/libblkid.so.1"), 0x00007725f5032000},
		}, []*check.Absolute{
			check.MustAbs("/lib64"),
			check.MustAbs("/usr/lib"),
			check.MustAbs("/usr/lib64"),
		}},
	}
	for _, tc := range testCases {
		t.Run(tc.file, func(t *testing.T) {
			t.Parallel()

			if got, err := ldd.Parse([]byte(tc.out)); err != nil {
				t.Errorf("Parse: error = %v", err)
			} else if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Parse: \n%s\nwant\n%s", mustMarshalJSON(got), mustMarshalJSON(tc.want))
			} else if paths := ldd.Path(got); !reflect.DeepEqual(paths, tc.paths) {
				t.Errorf("Paths: %v, want %v", paths, tc.paths)
			}
		})
	}
}

func TestString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		e    ldd.Entry
		want string
	}{
		{"ld", ldd.Entry{
			Name:     "/lib/ld-musl-x86_64.so.1",
			Location: 0x7ff71c0a4000,
		}, `/lib/ld-musl-x86_64.so.1 (0x7ff71c0a4000)`},

		{"libzstd", ldd.Entry{
			Name:     "libzstd.so.1",
			Path:     check.MustAbs("/usr/lib/libzstd.so.1"),
			Location: 0x7ff71bfd2000,
		}, `libzstd.so.1 => /usr/lib/libzstd.so.1 (0x7ff71bfd2000)`},

		{"liblzma", ldd.Entry{
			Name:     "liblzma.so.5",
			Path:     check.MustAbs("/usr/lib/liblzma.so.5"),
			Location: 0x7ff71bf9a000,
		}, `liblzma.so.5 => /usr/lib/liblzma.so.5 (0x7ff71bf9a000)`},

		{"libz", ldd.Entry{
			Name:     "libz.so.1",
			Path:     check.MustAbs("/lib/libz.so.1"),
			Location: 0x7ff71bf80000,
		}, `libz.so.1 => /lib/libz.so.1 (0x7ff71bf80000)`},

		{"libcrypto", ldd.Entry{
			Name:     "libcrypto.so.3",
			Path:     check.MustAbs("/lib/libcrypto.so.3"),
			Location: 0x7ff71ba00000,
		}, `libcrypto.so.3 => /lib/libcrypto.so.3 (0x7ff71ba00000)`},

		{"libc", ldd.Entry{
			Name:     "libc.musl-x86_64.so.1",
			Path:     check.MustAbs("/lib/ld-musl-x86_64.so.1"),
			Location: 0x7ff71c0a4000,
		}, `libc.musl-x86_64.so.1 => /lib/ld-musl-x86_64.so.1 (0x7ff71c0a4000)`},

		{"invalid", ldd.Entry{
			Location: 0x7ff71c0a4000,
		}, `invalid (0x7ff71c0a4000)`},

		{"invalid long", ldd.Entry{
			Path:     check.MustAbs("/lib/ld-musl-x86_64.so.1"),
			Location: 0x7ff71c0a4000,
		}, `invalid => /lib/ld-musl-x86_64.so.1 (0x7ff71c0a4000)`},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			t.Run("decode", func(t *testing.T) {
				if tc.e.Name == "" {
					return
				}
				t.Parallel()

				var got ldd.Entry
				if err := got.UnmarshalText([]byte(tc.want)); err != nil {
					t.Fatalf("UnmarshalText: error = %v", err)
				}

				if !reflect.DeepEqual(&got, &tc.e) {
					t.Errorf("UnmarshalText: %#v, want %#v", got, tc.e)
				}
			})

			t.Run("encode", func(t *testing.T) {
				t.Parallel()

				if got := tc.e.String(); got != tc.want {
					t.Errorf("String: %s, want %s", got, tc.want)
				}
			})
		})
	}
}

// mustMarshalJSON calls [json.Marshal] and returns the resulting data.
func mustMarshalJSON(v any) []byte {
	if data, err := json.Marshal(v); err != nil {
		panic(err)
	} else {
		return data
	}
}
