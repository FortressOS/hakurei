package app

import (
	"crypto/rand"
	"encoding/hex"
)

type ID [16]byte

func (a *ID) String() string {
	return hex.EncodeToString(a[:])
}

func newAppID(id *ID) error {
	_, err := rand.Read(id[:])
	return err
}
