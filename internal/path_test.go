package internal

import (
	"reflect"
	"testing"

	"hakurei.app/container/check"
)

func TestMustCheckPath(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		pathname  string
		wantFatal string
	}{
		{"poison", compPoison, "invalid test path, this program is compiled incorrectly"},
		{"zero", "", "invalid test path, this program is compiled incorrectly"},
		{"not absolute", "\x00", `path "\x00" is not absolute`},
		{"success", "/proc/nonexistent", ""},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fatal := func(v ...any) { t.Fatal(append([]any{"invalid call to fatal:"}, v...)...) }
			if tc.wantFatal != "" {
				fatal = func(v ...any) {
					if len(v) != 1 {
						t.Errorf("mustCheckPath: fatal %#v", v)
					} else if gotFatal, ok := v[0].(string); !ok {
						t.Errorf("mustCheckPath: fatal = %#v", v[0])
					} else if gotFatal != tc.wantFatal {
						t.Errorf("mustCheckPath: fatal = %q, want %q", gotFatal, tc.wantFatal)
					}

					// do not simulate exit
				}
			}

			if got := mustCheckPath(fatal, "test", tc.pathname); got != nil && !reflect.DeepEqual(got, check.MustAbs(tc.pathname)) {
				t.Errorf("mustCheckPath: %q", got)
			}
		})
	}
}
