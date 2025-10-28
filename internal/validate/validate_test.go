package validate_test

import (
	"testing"

	"hakurei.app/internal/validate"
)

func TestDeepContainsH(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		basepath string
		targpath string
		want     bool
		wantErr  bool
	}{
		{
			name: "empty",
			want: true,
		},
		{
			name:     "equal abs",
			basepath: "/run",
			targpath: "/run",
			want:     true,
		},
		{
			name:     "equal rel",
			basepath: "./run",
			targpath: "run",
			want:     true,
		},
		{
			name:     "contains abs",
			basepath: "/run",
			targpath: "/run/dbus",
			want:     true,
		},
		{
			name:     "inverse contains abs",
			basepath: "/run/dbus",
			targpath: "/run",
			want:     false,
		},
		{
			name:     "contains rel",
			basepath: "../run",
			targpath: "../run/dbus",
			want:     true,
		},
		{
			name:     "inverse contains rel",
			basepath: "../run/dbus",
			targpath: "../run",
			want:     false,
		},
		{
			name:     "weird abs",
			basepath: "/run/dbus",
			targpath: "/run/dbus/../current-system",
			want:     false,
		},
		{
			name:     "weird rel",
			basepath: "../run/dbus",
			targpath: "../run/dbus/../current-system",
			want:     false,
		},

		{
			name:     "invalid mix",
			basepath: "/run",
			targpath: "./run",
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got, err := validate.DeepContainsH(tc.basepath, tc.targpath); (err != nil) != tc.wantErr {
				t.Errorf("DeepContainsH: error = %v, wantErr %v", err, tc.wantErr)
			} else if got != tc.want {
				t.Errorf("DeepContainsH: = %v, want %v", got, tc.want)
			}
		})
	}
}
