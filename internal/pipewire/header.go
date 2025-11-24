package pipewire

import (
	"encoding/binary"
	"errors"
)

const (
	// HeaderSize is the fixed size of [Header].
	HeaderSize = 16
	// SizeMax is the largest value of [Header.Size] that can be represented in its 3-byte segment.
	SizeMax = 0x00ffffff
)

var (
	// ErrSizeRange indicates that the value of [Header.Size] cannot be represented in its 3-byte segment.
	ErrSizeRange = errors.New("size out of range")
	// ErrBadHeader indicates that the header slice does not have length [HeaderSize].
	ErrBadHeader = errors.New("incorrect header size")
)

// A Header is the fixed-size message header described in protocol native.
type Header struct {
	// The message id this is the destination resource/proxy id.
	ID Word `json:"Id"`
	// The opcode on the resource/proxy interface.
	Opcode byte `json:"opcode"`
	// The size of the payload and optional footer of the message.
	// Note: this value is only 24 bits long in the format.
	Size uint32 `json:"size"`
	// An increasing sequence number for each message.
	Sequence Word `json:"seq"`
	// Number of file descriptors in this message.
	FileCount Word `json:"n_fds"`
}

// append appends the protocol native message header to data.
//
// Callers must perform bounds check on [Header.Size].
func (h *Header) append(data []byte) []byte {
	data = binary.NativeEndian.AppendUint32(data, h.ID)
	data = binary.NativeEndian.AppendUint32(data, Word(h.Opcode)<<24|h.Size)
	data = binary.NativeEndian.AppendUint32(data, h.Sequence)
	data = binary.NativeEndian.AppendUint32(data, h.FileCount)
	return data
}

// MarshalBinary encodes the protocol native message header.
func (h *Header) MarshalBinary() (data []byte, err error) {
	if h.Size&^SizeMax != 0 {
		return nil, ErrSizeRange
	}
	return h.append(make([]byte, 0, HeaderSize)), nil
}

// unmarshalBinary decodes the protocol native message header.
func (h *Header) unmarshalBinary(data [HeaderSize]byte) {
	h.ID = binary.NativeEndian.Uint32(data[0:4])
	h.Size = binary.NativeEndian.Uint32(data[4:8])
	h.Opcode = byte(h.Size >> 24)
	h.Size &= SizeMax
	h.Sequence = binary.NativeEndian.Uint32(data[8:])
	h.FileCount = binary.NativeEndian.Uint32(data[12:])
}

// UnmarshalBinary decodes the protocol native message header.
func (h *Header) UnmarshalBinary(data []byte) error {
	if len(data) != HeaderSize {
		return ErrBadHeader
	}
	h.unmarshalBinary(([HeaderSize]byte)(data))
	return nil
}
