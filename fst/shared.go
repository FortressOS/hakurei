// Package fst exports shared fortify types.
package fst

import (
	"crypto/rand"
	"encoding/hex"
)

type ID [16]byte

func (a *ID) String() string {
	return hex.EncodeToString(a[:])
}

func NewAppID(id *ID) error {
	_, err := rand.Read(id[:])
	return err
}
