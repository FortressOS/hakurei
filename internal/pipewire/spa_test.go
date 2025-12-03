package pipewire_test

import (
	_ "embed"
	"encoding/binary"
	"testing"

	"hakurei.app/internal/pipewire"
)

func TestSPAKind(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		v    pipewire.SPAKind
		want string
	}{
		{"invalid", 0xdeadbeef, "invalid type field 0xdeadbeef"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := tc.v.String(); got != tc.want {
				t.Errorf("String: %q, want %q", got, tc.want)
			}
		})
	}
}

// splitMessages splits concatenated messages into groups of
// header, payload, footer of each individual message.
// splitMessages panics on any decoding error.
func splitMessages(iovec string) (messages [][3][]byte) {
	data := []byte(iovec)
	messages = make([][3][]byte, 0, 1<<7)

	var header pipewire.Header
	for len(data) != 0 {
		if err := header.UnmarshalBinary(data[:pipewire.SizeHeader]); err != nil {
			panic(err)
		}
		size := pipewire.SizePrefix + binary.NativeEndian.Uint32(data[pipewire.SizeHeader:])
		messages = append(messages, [3][]byte{
			data[:pipewire.SizeHeader],
			data[pipewire.SizeHeader : pipewire.SizeHeader+size],
			data[pipewire.SizeHeader+size : pipewire.SizeHeader+header.Size],
		})
		data = data[pipewire.SizeHeader+header.Size:]
	}
	return
}

var (
	//go:embed testdata/pw-container-00-sendmsg
	samplePWContainer00 string
	//go:embed testdata/pw-container-01-recvmsg
	samplePWContainer01 string
	//go:embed testdata/pw-container-03-sendmsg
	samplePWContainer03 string
	//go:embed testdata/pw-container-04-recvmsg
	samplePWContainer04 string
	//go:embed testdata/pw-container-06-sendmsg
	samplePWContainer06 string
	//go:embed testdata/pw-container-07-recvmsg
	samplePWContainer07 string

	// samplePWContainer is a collection of messages from the pw-container sample.
	samplePWContainer = [...][][3][]byte{
		splitMessages(samplePWContainer00),
		splitMessages(samplePWContainer01),
		nil,
		splitMessages(samplePWContainer03),
		splitMessages(samplePWContainer04),
		nil,
		splitMessages(samplePWContainer06),
		splitMessages(samplePWContainer07),
		nil,
	}
)
