package pipewire_test

import (
	"reflect"
	"testing"

	"hakurei.app/internal/pipewire"
)

func TestHeader(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		data []byte
		want pipewire.Header
	}{
		{"PW_SECURITY_CONTEXT_METHOD_CREATE", []byte{
			// Id
			3, 0, 0, 0,
			// size
			0xd8, 0, 0,
			// opcode
			1,
			// seq
			5, 0, 0, 0,
			// n_fds
			2, 0, 0, 0,
		}, pipewire.Header{ID: 3, Opcode: pipewire.PW_SECURITY_CONTEXT_METHOD_CREATE,
			Size: 0xd8, Sequence: 5, FileCount: 2}},

		{"PW_SECURITY_CONTEXT_METHOD_NUM", []byte{
			// Id
			0, 0, 0, 0,
			// size
			0x28, 0, 0,
			// opcode
			2,
			// seq
			6, 0, 0, 0,
			// n_fds
			0, 0, 0, 0,
		}, pipewire.Header{ID: 0, Opcode: pipewire.PW_SECURITY_CONTEXT_METHOD_NUM,
			Size: 0x28, Sequence: 6, FileCount: 0}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			t.Run("decode", func(t *testing.T) {
				t.Parallel()

				var got pipewire.Header
				if err := got.UnmarshalBinary(tc.data); err != nil {
					t.Fatalf("UnmarshalBinary: error = %v", err)
				}
				if got != tc.want {
					t.Fatalf("UnmarshalBinary: %#v, want %#v", got, tc.want)
				}
			})

			t.Run("encode", func(t *testing.T) {
				t.Parallel()

				if got, err := tc.want.MarshalBinary(); err != nil {
					t.Fatalf("MarshalBinary: error = %v", err)
				} else if string(got) != string(tc.data) {
					t.Fatalf("MarshalBinary: %#v, want %#v", got, tc.data)
				}
			})
		})
	}

	t.Run("size range", func(t *testing.T) {
		t.Parallel()

		if _, err := (&pipewire.Header{Size: 0xff000000}).MarshalBinary(); !reflect.DeepEqual(err, pipewire.ErrSizeRange) {
			t.Errorf("UnmarshalBinary: error = %v", err)
		}
	})

	t.Run("header size", func(t *testing.T) {
		t.Parallel()

		if err := (*pipewire.Header)(nil).UnmarshalBinary(nil); !reflect.DeepEqual(err, pipewire.ErrBadHeader) {
			t.Errorf("UnmarshalBinary: error = %v", err)
		}
	})
}
