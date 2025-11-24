package pipewire_test

import (
	"reflect"
	"testing"

	"hakurei.app/internal/pipewire"
)

func TestHeader(t *testing.T) {
	t.Parallel()

	encodingTestCases[pipewire.Header, *pipewire.Header]{
		{"PW_CORE_METHOD_HELLO", []byte{
			// Id
			0, 0, 0, 0,
			// size
			0x18, 0, 0,
			// opcode
			1,
			// seq
			0, 0, 0, 0,
			// n_fds
			0, 0, 0, 0,
		}, pipewire.Header{ID: pipewire.PW_ID_CORE, Opcode: pipewire.PW_CORE_METHOD_HELLO,
			Size: 0x18, Sequence: 0, FileCount: 0}, nil},

		{"PW_CLIENT_METHOD_UPDATE_PROPERTIES", []byte{
			// Id
			1, 0, 0, 0,
			// size
			0, 6, 0,
			// opcode
			2,
			// seq
			1, 0, 0, 0,
			// n_fds
			0, 0, 0, 0,
		}, pipewire.Header{ID: pipewire.PW_ID_CLIENT, Opcode: pipewire.PW_CLIENT_METHOD_UPDATE_PROPERTIES,
			Size: 0x600, Sequence: 1, FileCount: 0}, nil},

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
			Size: 0xd8, Sequence: 5, FileCount: 2}, nil},

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
			Size: 0x28, Sequence: 6, FileCount: 0}, nil},
	}.run(t)

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
