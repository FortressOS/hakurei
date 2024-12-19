package fst

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
)

type ID [16]byte

var (
	ErrInvalidLength = errors.New("string representation must have a length of 32")
)

func (a *ID) String() string {
	return hex.EncodeToString(a[:])
}

func NewAppID(id *ID) error {
	_, err := rand.Read(id[:])
	return err
}

func ParseAppID(id *ID, s string) error {
	if len(s) != 32 {
		return ErrInvalidLength
	}

	for i, b := range s {
		if b < '0' || b > 'f' {
			return fmt.Errorf("invalid char %q at byte %d", b, i)
		}

		v := uint8(b)
		if v > '9' {
			v = 10 + v - 'a'
		} else {
			v -= '0'
		}
		if i%2 == 0 {
			v <<= 4
		}
		id[i/2] += v
	}

	return nil
}
