package ldd_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"git.gensokyo.uk/security/fortify/ldd"
)

func TestParseError(t *testing.T) {
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
`, ldd.ErrPathNotAbsolute},
		{"unexpected segments", `
meow libzstd.so.1 => /usr/lib/libzstd.so.1 (0x7ff71bfd2000)
`, ldd.EntryUnexpectedSegmentsError("meow libzstd.so.1 => /usr/lib/libzstd.so.1 (0x7ff71bfd2000)")},
		{"bad location format", `
libzstd.so.1 => /usr/lib/libzstd.so.1 7ff71bfd2000
`, ldd.ErrBadLocationFormat},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stdout := new(strings.Builder)
			stdout.WriteString(tc.out)

			if _, err := ldd.Parse(stdout); !errors.Is(err, tc.wantErr) {
				t.Errorf("Parse() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestParse(t *testing.T) {
	testCases := []struct {
		file, out string
		want      []*ldd.Entry
	}{
		{"musl /bin/kmod", `
/lib/ld-musl-x86_64.so.1 (0x7ff71c0a4000)
libzstd.so.1 => /usr/lib/libzstd.so.1 (0x7ff71bfd2000)
liblzma.so.5 => /usr/lib/liblzma.so.5 (0x7ff71bf9a000)
libz.so.1 => /lib/libz.so.1 (0x7ff71bf80000)
libcrypto.so.3 => /lib/libcrypto.so.3 (0x7ff71ba00000)
libc.musl-x86_64.so.1 => /lib/ld-musl-x86_64.so.1 (0x7ff71c0a4000)`,
			[]*ldd.Entry{
				{"/lib/ld-musl-x86_64.so.1", "", 0x7ff71c0a4000},
				{"libzstd.so.1", "/usr/lib/libzstd.so.1", 0x7ff71bfd2000},
				{"liblzma.so.5", "/usr/lib/liblzma.so.5", 0x7ff71bf9a000},
				{"libz.so.1", "/lib/libz.so.1", 0x7ff71bf80000},
				{"libcrypto.so.3", "/lib/libcrypto.so.3", 0x7ff71ba00000},
				{"libc.musl-x86_64.so.1", "/lib/ld-musl-x86_64.so.1", 0x7ff71c0a4000},
			}},
		{"glibc /nix/store/rc3n2r3nffpib2gqpxlkjx36frw6n34z-kmod-31/bin/kmod", `
	linux-vdso.so.1 (0x00007ffed65be000)
	libzstd.so.1 => /nix/store/80pxmvb9q43kh9rkjagc4h41vf6dh1y6-zstd-1.5.6/lib/libzstd.so.1 (0x00007f3199cd1000)
	liblzma.so.5 => /nix/store/g78jna1i5qhh8gqs4mr64648f0szqgw4-xz-5.4.7/lib/liblzma.so.5 (0x00007f3199ca2000)
	libc.so.6 => /nix/store/c10zhkbp6jmyh0xc5kd123ga8yy2p4hk-glibc-2.39-52/lib/libc.so.6 (0x00007f3199ab5000)
	libpthread.so.0 => /nix/store/c10zhkbp6jmyh0xc5kd123ga8yy2p4hk-glibc-2.39-52/lib/libpthread.so.0 (0x00007f3199ab0000)
	/nix/store/c10zhkbp6jmyh0xc5kd123ga8yy2p4hk-glibc-2.39-52/lib/ld-linux-x86-64.so.2 => /nix/store/c10zhkbp6jmyh0xc5kd123ga8yy2p4hk-glibc-2.39-52/lib64/ld-linux-x86-64.so.2 (0x00007f3199da5000)`,
			[]*ldd.Entry{
				{"linux-vdso.so.1", "", 0x00007ffed65be000},
				{"libzstd.so.1", "/nix/store/80pxmvb9q43kh9rkjagc4h41vf6dh1y6-zstd-1.5.6/lib/libzstd.so.1", 0x00007f3199cd1000},
				{"liblzma.so.5", "/nix/store/g78jna1i5qhh8gqs4mr64648f0szqgw4-xz-5.4.7/lib/liblzma.so.5", 0x00007f3199ca2000},
				{"libc.so.6", "/nix/store/c10zhkbp6jmyh0xc5kd123ga8yy2p4hk-glibc-2.39-52/lib/libc.so.6", 0x00007f3199ab5000},
				{"libpthread.so.0", "/nix/store/c10zhkbp6jmyh0xc5kd123ga8yy2p4hk-glibc-2.39-52/lib/libpthread.so.0", 0x00007f3199ab0000},
				{"/nix/store/c10zhkbp6jmyh0xc5kd123ga8yy2p4hk-glibc-2.39-52/lib/ld-linux-x86-64.so.2", "/nix/store/c10zhkbp6jmyh0xc5kd123ga8yy2p4hk-glibc-2.39-52/lib64/ld-linux-x86-64.so.2", 0x00007f3199da5000},
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
	libblkid.so.1 => /usr/lib/libblkid.so.1 (0x00007725f5032000)`,
			[]*ldd.Entry{
				{"linux-vdso.so.1", "", 0x00007725f5772000},
				{"libglib-2.0.so.0", "/usr/lib/libglib-2.0.so.0", 0x00007725f55d5000},
				{"libgio-2.0.so.0", "/usr/lib/libgio-2.0.so.0", 0x00007725f5406000},
				{"libgobject-2.0.so.0", "/usr/lib/libgobject-2.0.so.0", 0x00007725f53a6000},
				{"libgcc_s.so.1", "/usr/lib/libgcc_s.so.1", 0x00007725f5378000},
				{"libc.so.6", "/usr/lib/libc.so.6", 0x00007725f5187000},
				{"libpcre2-8.so.0", "/usr/lib/libpcre2-8.so.0", 0x00007725f50e8000},
				{"libgmodule-2.0.so.0", "/usr/lib/libgmodule-2.0.so.0", 0x00007725f50df000},
				{"libz.so.1", "/usr/lib/libz.so.1", 0x00007725f50c6000},
				{"libmount.so.1", "/usr/lib/libmount.so.1", 0x00007725f5076000},
				{"libffi.so.8", "/usr/lib/libffi.so.8", 0x00007725f506b000},
				{"/lib64/ld-linux-x86-64.so.2", "/usr/lib64/ld-linux-x86-64.so.2", 0x00007725f5774000},
				{"libblkid.so.1", "/usr/lib/libblkid.so.1", 0x00007725f5032000},
			}},
	}
	for _, tc := range testCases {
		t.Run(tc.file, func(t *testing.T) {
			stdout := new(strings.Builder)
			stdout.WriteString(tc.out)

			if got, err := ldd.Parse(stdout); err != nil {
				t.Errorf("Parse() error = %v", err)
			} else if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Parse() got = %#v, want %#v", got, tc.want)
			}
		})
	}
}
