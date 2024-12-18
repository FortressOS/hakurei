package proc

import (
	"encoding/gob"
	"errors"
	"os"
	"strconv"
)

var (
	ErrNotSet  = errors.New("environment variable not set")
	ErrInvalid = errors.New("bad file descriptor")
)

func Receive(key string, e any) (func() error, error) {
	var setup *os.File

	if s, ok := os.LookupEnv(key); !ok {
		return nil, ErrNotSet
	} else {
		if fd, err := strconv.Atoi(s); err != nil {
			return nil, err
		} else {
			setup = os.NewFile(uintptr(fd), "setup")
			if setup == nil {
				return nil, ErrInvalid
			}
		}
	}

	return func() error { return setup.Close() }, gob.NewDecoder(setup).Decode(e)
}
