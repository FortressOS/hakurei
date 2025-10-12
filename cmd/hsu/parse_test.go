package main

import (
	"bytes"
	"strconv"
	"testing"
)

func TestParseUint32Fast(t *testing.T) {
	t.Parallel()

	t.Run("zero-length", func(t *testing.T) {
		t.Parallel()

		if _, err := parseUint32Fast(""); err == nil || err.Error() != "zero length string" {
			t.Errorf(`parseUint32Fast(""): error = %v`, err)
			return
		}
	})

	t.Run("overflow", func(t *testing.T) {
		t.Parallel()

		if _, err := parseUint32Fast("10000000000"); err == nil || err.Error() != "string too long" {
			t.Errorf("parseUint32Fast: error = %v", err)
			return
		}
	})

	t.Run("invalid byte", func(t *testing.T) {
		t.Parallel()

		if _, err := parseUint32Fast("meow"); err == nil || err.Error() != "invalid character 'm' at index 0" {
			t.Errorf(`parseUint32Fast("meow"): error = %v`, err)
			return
		}
	})

	t.Run("full range", func(t *testing.T) {
		t.Parallel()

		testRange := func(i, end int) {
			for ; i < end; i++ {
				s := strconv.Itoa(i)
				w := i
				t.Run("parse "+s, func(t *testing.T) {
					t.Parallel()

					v, err := parseUint32Fast(s)
					if err != nil {
						t.Errorf("parseUint32Fast(%q): error = %v",
							s, err)
						return
					}
					if v != w {
						t.Errorf("parseUint32Fast(%q): got %v",
							s, v)
						return
					}
				})
			}
		}

		testRange(0, 5000)
		testRange(105000, 110000)
		testRange(23005000, 23010000)
		testRange(456005000, 456010000)
		testRange(7890005000, 7890010000)
	})
}

func TestParseConfig(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		puid, want int
		wantErr    string
		rc         string
	}{
		{"empty", 0, -1, "", ``},
		{"invalid field", 0, -1, "invalid entry on line 1", `9`},
		{"invalid puid", 0, -1, "invalid parent uid on line 1", `f 9`},
		{"invalid fid", 1000, -1, "invalid identity on line 1", `1000 f`},
		{"match", 1000, 0, "", `1000 0`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fid, ok, err := parseConfig(bytes.NewBufferString(tc.rc), tc.puid)
			if err == nil && tc.wantErr != "" {
				t.Errorf("parseConfig: error = %v; wantErr %q",
					err, tc.wantErr)
				return
			}
			if err != nil && err.Error() != tc.wantErr {
				t.Errorf("parseConfig: error = %q; wantErr %q",
					err, tc.wantErr)
				return
			}
			if ok == (tc.want == -1) {
				t.Errorf("parseConfig: ok = %v; want %v",
					ok, tc.want)
				return
			}
			if fid != tc.want {
				t.Errorf("parseConfig: fid = %v; want %v",
					fid, tc.want)
			}
		})
	}
}
