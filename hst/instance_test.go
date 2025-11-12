package hst_test

import (
	"encoding/hex"
	"errors"
	"reflect"
	"testing"
	"time"
	_ "unsafe" // for go:linkname

	"hakurei.app/hst"
)

// Made available here to check time encoding behaviour of [hst.ID].
//
//go:linkname newInstanceID hakurei.app/hst.newInstanceID
func newInstanceID(id *hst.ID, p uint64) error

func TestIdentifierDecodeError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		err  error
		want string
	}{
		{"invalid byte", hst.IdentifierDecodeError{Err: hex.InvalidByteError(0)},
			"got invalid byte U+0000 in identifier"},
		{"odd length", hst.IdentifierDecodeError{Err: hex.ErrLength},
			"odd length identifier hex string"},
		{"passthrough", hst.IdentifierDecodeError{Err: hst.ErrIdentifierLength},
			hst.ErrIdentifierLength.Error()},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := tc.err.Error(); got != tc.want {
				t.Errorf("Error: %q, want %q", got, tc.want)
			}
		})
	}

	t.Run("unwrap", func(t *testing.T) {
		t.Parallel()

		err := hst.IdentifierDecodeError{Err: hst.ErrIdentifierLength}
		if !errors.Is(err, hst.ErrIdentifierLength) {
			t.Errorf("Is unexpected false")
		}
	})
}

func TestID(t *testing.T) {
	t.Parallel()

	var randomID hst.ID
	if err := hst.NewInstanceID(&randomID); err != nil {
		t.Fatalf("NewInstanceID: error = %v", err)
	}

	testCases := []struct {
		name string
		data string
		want hst.ID
		err  error
	}{
		{"bad length", "meow", hst.ID{},
			hst.IdentifierDecodeError{Err: hst.ErrIdentifierLength}},
		{"invalid byte", "02bc7f8936b2af6\x00\x00e2535cd71ef0bb7", hst.ID{},
			hst.IdentifierDecodeError{Err: hex.InvalidByteError(0)}},

		{"zero", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", hst.ID{}, nil},
		{"random", randomID.String(), randomID, nil},
		{"sample", "ba21c9bd33d9d37917288281a2a0d239", hst.ID{
			0xba, 0x21, 0xc9, 0xbd,
			0x33, 0xd9, 0xd3, 0x79,
			0x17, 0x28, 0x82, 0x81,
			0xa2, 0xa0, 0xd2, 0x39}, nil},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var got hst.ID
			if err := got.UnmarshalText([]byte(tc.data)); !reflect.DeepEqual(err, tc.err) {
				t.Errorf("UnmarshalText: error = %#v, want %#v", err, tc.err)
			}

			if tc.err == nil {
				if gotString := got.String(); gotString != tc.data {
					t.Errorf("String: %q, want %q", gotString, tc.data)
				}
				if gotData, _ := got.MarshalText(); string(gotData) != tc.data {
					t.Errorf("MarshalText: %q, want %q", string(gotData), tc.data)
				}
			}
		})
	}

	t.Run("time", func(t *testing.T) {
		t.Parallel()
		var id hst.ID

		now := time.Now()
		if err := newInstanceID(&id, uint64(now.UnixNano())); err != nil {
			t.Fatalf("newInstanceID: error = %v", err)
		}

		got := id.CreationTime()
		if !got.Equal(now) {
			t.Fatalf("CreationTime(%q): %s, want %s", id.String(), got, now)
		}
	})
}
