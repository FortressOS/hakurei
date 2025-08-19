package container

import (
	"os"
	"slices"
	"testing"
)

func TestParentPerm(t *testing.T) {
	testCases := []struct {
		perm os.FileMode
		want os.FileMode
	}{
		{0755, 0755},
		{0750, 0750},
		{0705, 0705},
		{0700, 0700},
		{050, 0750},
		{05, 0705},
		{0, 0700},
	}

	for _, tc := range testCases {
		t.Run(tc.perm.String(), func(t *testing.T) {
			if got := parentPerm(tc.perm); got != tc.want {
				t.Errorf("parentPerm: %#o, want %#o", got, tc.want)
			}
		})
	}
}

func TestEscapeOverlayDataSegment(t *testing.T) {
	testCases := []struct {
		name string
		s    string
		want string
	}{
		{"zero", zeroString, zeroString},
		{"multi", `\\\:,:,\\\`, `\\\\\\\:\,\:\,\\\\\\`},
		{"bwrap", `/path :,\`, `/path \:\,\\`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := EscapeOverlayDataSegment(tc.s); got != tc.want {
				t.Errorf("escapeOverlayDataSegment: %s, want %s", got, tc.want)
			}
		})
	}
}

type opsBuilderTestCase struct {
	name string
	ops  *Ops
	want Ops
}

func checkOpsBuilder(t *testing.T, testCases []opsBuilderTestCase) {
	t.Run("build", func(t *testing.T) {
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				if !slices.EqualFunc(*tc.ops, tc.want, func(op Op, v Op) bool { return op.Is(v) }) {
					t.Errorf("Ops: %#v, want %#v", tc.ops, tc.want)
				}
			})
		}
	})
}

type opIsTestCase struct {
	name  string
	op, v Op
	want  bool
}

func checkOpIs(t *testing.T, testCases []opIsTestCase) {
	t.Run("is", func(t *testing.T) {
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				if got := tc.op.Is(tc.v); got != tc.want {
					t.Errorf("Is: %v, want %v", got, tc.want)
				}
			})
		}
	})
}

type opMetaTestCase struct {
	name string
	op   Op

	wantPrefix string
	wantString string
}

func checkOpMeta(t *testing.T, testCases []opMetaTestCase) {
	t.Run("meta", func(t *testing.T) {
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Run("prefix", func(t *testing.T) {
					if got := tc.op.prefix(); got != tc.wantPrefix {
						t.Errorf("prefix: %q, want %q", got, tc.wantPrefix)
					}
				})

				t.Run("string", func(t *testing.T) {
					if got := tc.op.String(); got != tc.wantString {
						t.Errorf("String: %s, want %s", got, tc.wantString)
					}
				})
			})
		}
	})
}
