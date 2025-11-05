package env_test

import (
	"fmt"
	"reflect"
	"testing"

	"hakurei.app/container"
	"hakurei.app/container/check"
	"hakurei.app/container/fhs"
	"hakurei.app/container/stub"
	"hakurei.app/hst"
	"hakurei.app/internal/env"
)

func TestPaths(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		env  *env.Paths
		want hst.Paths

		wantPanic string
	}{
		{"nil", nil, hst.Paths{}, "attempting to use an invalid Paths"},
		{"zero", new(env.Paths), hst.Paths{}, "attempting to use an invalid Paths"},

		{"nil tempdir", &env.Paths{
			RuntimePath: fhs.AbsTmp,
		}, hst.Paths{}, "attempting to use an invalid Paths"},

		{"nil runtime", &env.Paths{
			TempDir: fhs.AbsTmp,
		}, hst.Paths{
			TempDir:     fhs.AbsTmp,
			SharePath:   fhs.AbsTmp.Append("hakurei.57005"),
			RuntimePath: fhs.AbsTmp.Append("hakurei.57005/compat"),
			RunDirPath:  fhs.AbsTmp.Append("hakurei.57005/compat/hakurei"),
		}, ""},

		{"full", &env.Paths{
			TempDir:     fhs.AbsTmp,
			RuntimePath: fhs.AbsRunUser.Append("1000"),
		}, hst.Paths{
			TempDir:     fhs.AbsTmp,
			SharePath:   fhs.AbsTmp.Append("hakurei.57005"),
			RuntimePath: fhs.AbsRunUser.Append("1000"),
			RunDirPath:  fhs.AbsRunUser.Append("1000/hakurei"),
		}, ""},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.wantPanic != "" {
				defer func() {
					if r := recover(); r != tc.wantPanic {
						t.Errorf("Copy: panic = %#v, want %q", r, tc.wantPanic)
					}
				}()
			}

			var sc hst.Paths
			tc.env.Copy(&sc, 0xdead)
			if !reflect.DeepEqual(&sc, &tc.want) {
				t.Errorf("Copy: %#v, want %#v", sc, tc.want)
			}
		})
	}
}

func TestCopyPaths(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		env   map[string]string
		tmp   string
		fatal string
		want  env.Paths
	}{
		{"invalid tempdir", nil, "\x00",
			"invalid TMPDIR: path \"\\x00\" is not absolute", env.Paths{}},
		{"empty environment", make(map[string]string), container.Nonexistent,
			"", env.Paths{TempDir: check.MustAbs(container.Nonexistent)}},
		{"invalid XDG_RUNTIME_DIR", map[string]string{"XDG_RUNTIME_DIR": "\x00"}, container.Nonexistent,
			"", env.Paths{TempDir: check.MustAbs(container.Nonexistent)}},
		{"full", map[string]string{"XDG_RUNTIME_DIR": "/\x00"}, container.Nonexistent,
			"", env.Paths{TempDir: check.MustAbs(container.Nonexistent), RuntimePath: check.MustAbs("/\x00")}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.fatal != "" {
				defer stub.HandleExit(t)
			}

			got := env.CopyPathsFunc(func(format string, v ...any) {
				if tc.fatal == "" {
					t.Fatalf("unexpected call to fatalf: format = %q, v = %#v", format, v)
				}

				if got := fmt.Sprintf(format, v...); got != tc.fatal {
					t.Fatalf("fatalf: %q, want %q", got, tc.fatal)
				}
				panic(stub.PanicExit)
			}, func() string { return tc.tmp }, func(key string) string { return tc.env[key] })

			if tc.fatal != "" {
				t.Fatalf("copyPaths: expected fatal %q", tc.fatal)
			}

			if !reflect.DeepEqual(got, &tc.want) {
				t.Errorf("copyPaths: %#v, want %#v", got, &tc.want)
			}
		})
	}
}
