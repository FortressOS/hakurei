package pipewire_test

import (
	"reflect"
	"testing"

	"hakurei.app/internal/pipewire"
)

func TestHeader(t *testing.T) {
	t.Parallel()

	encodingTestCases[pipewire.Header, *pipewire.Header]{

		/* sendmsg 0 */

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

		/* recvmsg 0 */

		{"PW_CORE_EVENT_INFO", samplePWContainer[1][0][0], pipewire.Header{
			ID:     pipewire.PW_ID_CORE,
			Opcode: pipewire.PW_CORE_EVENT_INFO,
			Size:   0x6b8, Sequence: 0, FileCount: 0,
		}, nil},

		{"PW_CORE_EVENT_BOUND_PROPS 0", samplePWContainer[1][1][0], pipewire.Header{
			ID:     pipewire.PW_ID_CORE,
			Opcode: pipewire.PW_CORE_EVENT_BOUND_PROPS,
			Size:   0x198, Sequence: 1, FileCount: 0,
		}, nil},

		{"PW_CLIENT_EVENT_INFO 0", samplePWContainer[1][2][0], pipewire.Header{
			ID:     pipewire.PW_ID_CLIENT,
			Opcode: pipewire.PW_CLIENT_EVENT_INFO,
			Size:   0x1f0, Sequence: 2, FileCount: 0,
		}, nil},

		{"PW_CLIENT_EVENT_INFO 1", samplePWContainer[1][3][0], pipewire.Header{
			ID:     pipewire.PW_ID_CLIENT,
			Opcode: pipewire.PW_CLIENT_EVENT_INFO,
			Size:   0x7a0, Sequence: 3, FileCount: 0,
		}, nil},

		{"PW_CLIENT_EVENT_INFO 2", samplePWContainer[1][4][0], pipewire.Header{
			ID:     pipewire.PW_ID_CLIENT,
			Opcode: pipewire.PW_CLIENT_EVENT_INFO,
			Size:   0x7d0, Sequence: 4, FileCount: 0,
		}, nil},

		{"PW_CORE_EVENT_DONE 0", samplePWContainer[1][5][0], pipewire.Header{
			ID:     pipewire.PW_ID_CORE,
			Opcode: pipewire.PW_CORE_EVENT_DONE,
			Size:   0x58, Sequence: 5, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 0", samplePWContainer[1][6][0], pipewire.Header{
			ID:     2, // this is specified by Core::GetRegistry in samplePWContainer[0][2][1]
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0xc8, Sequence: 6, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 1", samplePWContainer[1][7][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0xd8, Sequence: 7, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 2", samplePWContainer[1][8][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0xa8, Sequence: 8, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 3", samplePWContainer[1][9][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0xe8, Sequence: 9, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 4", samplePWContainer[1][10][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0xa0, Sequence: 10, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 5", samplePWContainer[1][11][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0xe0, Sequence: 11, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 6", samplePWContainer[1][12][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0xe0, Sequence: 12, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 7", samplePWContainer[1][13][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0x170, Sequence: 13, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 8", samplePWContainer[1][14][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0xe8, Sequence: 14, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 9", samplePWContainer[1][15][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0x178, Sequence: 15, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 10", samplePWContainer[1][16][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0xe8, Sequence: 16, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 11", samplePWContainer[1][17][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0x170, Sequence: 17, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 12", samplePWContainer[1][18][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0xe0, Sequence: 18, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 13", samplePWContainer[1][19][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0x170, Sequence: 19, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 14", samplePWContainer[1][20][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0xe8, Sequence: 20, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 15", samplePWContainer[1][21][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0x170, Sequence: 21, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 16", samplePWContainer[1][22][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0xe0, Sequence: 22, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 17", samplePWContainer[1][23][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0xe0, Sequence: 23, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 18", samplePWContainer[1][24][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0xe0, Sequence: 24, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 19", samplePWContainer[1][25][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0x160, Sequence: 25, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 20", samplePWContainer[1][26][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0xe0, Sequence: 26, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 21", samplePWContainer[1][27][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0x168, Sequence: 27, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 22", samplePWContainer[1][28][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0xe8, Sequence: 28, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 23", samplePWContainer[1][29][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0x178, Sequence: 29, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 24", samplePWContainer[1][30][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0x178, Sequence: 30, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 25", samplePWContainer[1][31][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0x168, Sequence: 31, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 26", samplePWContainer[1][32][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0x170, Sequence: 32, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 27", samplePWContainer[1][33][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0x178, Sequence: 33, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 28", samplePWContainer[1][34][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0x170, Sequence: 34, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 29", samplePWContainer[1][35][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0xe0, Sequence: 35, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 30", samplePWContainer[1][36][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0xe8, Sequence: 36, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 31", samplePWContainer[1][37][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0x118, Sequence: 37, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 32", samplePWContainer[1][38][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0x120, Sequence: 38, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 33", samplePWContainer[1][39][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0xd0, Sequence: 39, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 34", samplePWContainer[1][40][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0x238, Sequence: 40, FileCount: 0,
		}, nil},

		{"PW_CORE_EVENT_DONE 1", samplePWContainer[1][41][0], pipewire.Header{
			ID:     pipewire.PW_ID_CORE,
			Opcode: pipewire.PW_CORE_EVENT_DONE,
			Size:   0x28, Sequence: 41, FileCount: 0,
		}, nil},

		{"PW_REGISTRY_EVENT_GLOBAL 35", samplePWContainer[1][42][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_EVENT_GLOBAL,
			Size:   0x268, Sequence: 42, FileCount: 0,
		}, nil},

		/* sendmsg 1 */

		{"PW_REGISTRY_METHOD_BIND", samplePWContainer[3][0][0], pipewire.Header{
			ID:     2,
			Opcode: pipewire.PW_REGISTRY_METHOD_BIND,
			Size:   0x98, Sequence: 4, FileCount: 0,
		}, nil},

		/* recvmsg 1 */

		{"PW_CORE_EVENT_BOUND_PROPS 1", samplePWContainer[4][0][0], pipewire.Header{
			ID:     pipewire.PW_ID_CORE,
			Opcode: pipewire.PW_CORE_EVENT_BOUND_PROPS,
			Size:   0x68, Sequence: 43, FileCount: 0,
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
