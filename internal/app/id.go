package app

import (
	"crypto/rand"
	"encoding/hex"
)

type appID [16]byte

func (a *appID) String() string {
	return hex.EncodeToString(a[:])
}

func newAppID() (*appID, error) {
	a := &appID{}
	_, err := rand.Read(a[:])
	return a, err
}
