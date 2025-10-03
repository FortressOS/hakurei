package app

import (
	"fmt"
	"reflect"
	"testing"

	"hakurei.app/container"
	"hakurei.app/container/stub"
	"hakurei.app/hst"
)

func TestEnvPaths(t *testing.T) {
	testCases := []struct {
		name string
		env  *EnvPaths
		want hst.Paths

		wantPanic string
	}{
		{"nil", nil, hst.Paths{}, "attempting to use an invalid EnvPaths"},
		{"zero", new(EnvPaths), hst.Paths{}, "attempting to use an invalid EnvPaths"},

		{"nil tempdir", &EnvPaths{
			RuntimePath: container.AbsFHSTmp,
		}, hst.Paths{}, "attempting to use an invalid EnvPaths"},

		{"nil runtime", &EnvPaths{
			TempDir: container.AbsFHSTmp,
		}, hst.Paths{
			TempDir:     container.AbsFHSTmp,
			SharePath:   container.AbsFHSTmp.Append("hakurei.3735928559"),
			RuntimePath: container.AbsFHSTmp.Append("hakurei.3735928559/run/compat"),
			RunDirPath:  container.AbsFHSTmp.Append("hakurei.3735928559/run"),
		}, ""},

		{"full", &EnvPaths{
			TempDir:     container.AbsFHSTmp,
			RuntimePath: container.AbsFHSRunUser.Append("1000"),
		}, hst.Paths{
			TempDir:     container.AbsFHSTmp,
			SharePath:   container.AbsFHSTmp.Append("hakurei.3735928559"),
			RuntimePath: container.AbsFHSRunUser.Append("1000"),
			RunDirPath:  container.AbsFHSRunUser.Append("1000/hakurei"),
		}, ""},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.wantPanic != "" {
				defer func() {
					if r := recover(); r != tc.wantPanic {
						t.Errorf("Copy: panic = %#v, want %q", r, tc.wantPanic)
					}
				}()
			}

			var sc hst.Paths
			tc.env.Copy(&sc, 0xdeadbeef)
			if !reflect.DeepEqual(&sc, &tc.want) {
				t.Errorf("Copy: %#v, want %#v", sc, tc.want)
			}
		})
	}
}

func TestCopyPaths(t *testing.T) {
	testCases := []struct {
		name  string
		env   map[string]string
		tmp   string
		fatal string
		want  EnvPaths
	}{
		{"invalid tempdir", nil, "\x00",
			"invalid TMPDIR: path \"\\x00\" is not absolute", EnvPaths{}},
		{"empty environment", make(map[string]string), container.Nonexistent,
			"", EnvPaths{TempDir: container.MustAbs(container.Nonexistent)}},
		{"invalid XDG_RUNTIME_DIR", map[string]string{"XDG_RUNTIME_DIR": "\x00"}, container.Nonexistent,
			"", EnvPaths{TempDir: container.MustAbs(container.Nonexistent)}},
		{"full", map[string]string{"XDG_RUNTIME_DIR": "/\x00"}, container.Nonexistent,
			"", EnvPaths{TempDir: container.MustAbs(container.Nonexistent), RuntimePath: container.MustAbs("/\x00")}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.fatal != "" {
				defer stub.HandleExit(t)
			}

			k := copyPathsDispatcher{t: t, env: tc.env, tmp: tc.tmp, expectsFatal: tc.fatal}
			got := copyPaths(k)

			if tc.fatal != "" {
				t.Fatalf("copyPaths: expected fatal %q", tc.fatal)
			}

			if !reflect.DeepEqual(got, &tc.want) {
				t.Errorf("copyPaths: %#v, want %#v", got, &tc.want)
			}
		})
	}
}

// copyPathsDispatcher implements enough of syscallDispatcher for all copyPaths code paths.
type copyPathsDispatcher struct {
	env map[string]string
	tmp string

	// must be checked at the conclusion of the test
	expectsFatal string

	t *testing.T
	panicDispatcher
}

func (k copyPathsDispatcher) tempdir() string { return k.tmp }
func (k copyPathsDispatcher) lookupEnv(key string) (value string, ok bool) {
	value, ok = k.env[key]
	return
}
func (k copyPathsDispatcher) fatalf(format string, v ...any) {
	if k.expectsFatal == "" {
		k.t.Fatalf("unexpected call to fatalf: format = %q, v = %#v", format, v)
	}

	if got := fmt.Sprintf(format, v...); got != k.expectsFatal {
		k.t.Fatalf("fatalf: %q, want %q", got, k.expectsFatal)
	}
	panic(stub.PanicExit)
}
