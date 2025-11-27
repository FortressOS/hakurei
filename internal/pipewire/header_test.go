package pipewire_test

import (
	"reflect"
	"testing"

	"hakurei.app/internal/pipewire"
)

func TestHeader(t *testing.T) {
	t.Parallel()

	encodingTestCases[pipewire.Header, *pipewire.Header]{
		{"PW_CORE_METHOD_HELLO", samplePWContainer[0][0][0], pipewire.Header{
			ID:     pipewire.PW_ID_CORE,
			Opcode: pipewire.PW_CORE_METHOD_HELLO,
			Size:   0x18, Sequence: 0, FileCount: 0,
		}, nil},

		{"PW_CLIENT_METHOD_UPDATE_PROPERTIES", samplePWContainer[0][1][0], pipewire.Header{
			ID:     pipewire.PW_ID_CLIENT,
			Opcode: pipewire.PW_CLIENT_METHOD_UPDATE_PROPERTIES,
			Size:   0x600, Sequence: 1, FileCount: 0,
		}, nil},

		{"PW_CORE_METHOD_GET_REGISTRY", samplePWContainer[0][2][0], pipewire.Header{
			ID:     pipewire.PW_ID_CORE,
			Opcode: pipewire.PW_CORE_METHOD_GET_REGISTRY,
			Size:   0x28, Sequence: 2, FileCount: 0,
		}, nil},

		{"PW_CORE_METHOD_SYNC", samplePWContainer[0][3][0], pipewire.Header{
			ID:     pipewire.PW_ID_CORE,
			Opcode: pipewire.PW_CORE_METHOD_SYNC,
			Size:   0x28, Sequence: 3, FileCount: 0,
		}, nil},

		{"PW_CORE_EVENT_INFO", samplePWContainer[1][0][0], pipewire.Header{
			ID:     pipewire.PW_ID_CORE,
			Opcode: pipewire.PW_CORE_EVENT_INFO,
			Size:   0x6b8, Sequence: 0, FileCount: 0,
		}, nil},

		{"PW_CORE_EVENT_BOUND_PROPS", samplePWContainer[1][1][0], pipewire.Header{
			ID:     pipewire.PW_ID_CORE,
			Opcode: pipewire.PW_CORE_EVENT_BOUND_PROPS,
			Size:   0x198, Sequence: 1, FileCount: 0,
		}, nil},

		{"PW_CLIENT_EVENT_INFO", samplePWContainer[1][2][0], pipewire.Header{
			ID:     pipewire.PW_ID_CLIENT,
			Opcode: pipewire.PW_CLIENT_EVENT_INFO,
			Size:   0x1f0, Sequence: 2, FileCount: 0,
		}, nil},

		{"PW_CLIENT_EVENT_INFO*", samplePWContainer[1][3][0], pipewire.Header{
			ID:     pipewire.PW_ID_CLIENT,
			Opcode: pipewire.PW_CLIENT_EVENT_INFO,
			Size:   0x7a0, Sequence: 3, FileCount: 0,
		}, nil},

		{"PW_CLIENT_EVENT_INFO**", samplePWContainer[1][4][0], pipewire.Header{
			ID:     pipewire.PW_ID_CLIENT,
			Opcode: pipewire.PW_CLIENT_EVENT_INFO,
			Size:   0x7d0, Sequence: 4, FileCount: 0,
		}, nil},

		{"PW_CORE_EVENT_DONE", samplePWContainer[1][5][0], pipewire.Header{
			ID:     pipewire.PW_ID_CORE,
			Opcode: pipewire.PW_CORE_EVENT_DONE,
			Size:   0x58, Sequence: 5, FileCount: 0,
		}, nil},

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
