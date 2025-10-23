package hst

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

// An ID is a unique identifier held by a running hakurei container.
type ID [16]byte

// ErrIdentifierLength is returned when encountering a [hex] representation of [ID] with unexpected length.
var ErrIdentifierLength = errors.New("identifier string has unexpected length")

// IdentifierDecodeError is returned by [ID.UnmarshalText] to provide relevant error descriptions.
type IdentifierDecodeError struct{ Err error }

func (e IdentifierDecodeError) Unwrap() error { return e.Err }
func (e IdentifierDecodeError) Error() string {
	var invalidByteError hex.InvalidByteError
	switch {
	case errors.As(e.Err, &invalidByteError):
		return fmt.Sprintf("got invalid byte %#U in identifier", rune(invalidByteError))
	case errors.Is(e.Err, hex.ErrLength):
		return "odd length identifier hex string"

	default:
		return e.Err.Error()
	}
}

// String returns the [hex] string representation of [ID].
func (a *ID) String() string { return hex.EncodeToString(a[:]) }

// CreationTime returns the point in time [ID] was created.
func (a *ID) CreationTime() time.Time {
	return time.Unix(0, int64(binary.BigEndian.Uint64(a[:8]))).UTC()
}

// NewInstanceID creates a new unique [ID].
func NewInstanceID(id *ID) error { return newInstanceID(id, uint64(time.Now().UnixNano())) }

// newInstanceID creates a new unique [ID] with the specified timestamp.
func newInstanceID(id *ID, p uint64) error {
	binary.BigEndian.PutUint64(id[:8], p)
	_, err := rand.Read(id[8:])
	return err
}

// MarshalText encodes the [hex] representation of [ID].
func (a *ID) MarshalText() (text []byte, err error) {
	text = make([]byte, hex.EncodedLen(len(a)))
	hex.Encode(text, a[:])
	return
}

// UnmarshalText decodes a [hex] representation of [ID].
func (a *ID) UnmarshalText(text []byte) error {
	dl := hex.DecodedLen(len(text))
	if dl != len(a) {
		return IdentifierDecodeError{ErrIdentifierLength}
	}
	_, err := hex.Decode(a[:], text)
	if err == nil {
		return nil
	}
	return IdentifierDecodeError{err}
}

// A State describes a running hakurei container.
type State struct {
	// Unique instance id, created by [NewInstanceID].
	ID ID `json:"instance"`
	// Monitoring process pid. Runs as the priv user.
	PID int `json:"pid"`
	// Shim process pid. Runs as the target user.
	ShimPID int `json:"shim_pid"`

	// Configuration used to start the container.
	Config *Config `json:"config"`

	// Point in time the shim process was created.
	Time time.Time `json:"time"`
}
