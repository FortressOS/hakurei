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

// Setup appends the read end of a pipe for payload transmission and returns its fd.
func Setup(extraFiles *[]*os.File) (int, *gob.Encoder, error) {
	if r, w, err := os.Pipe(); err != nil {
		return -1, nil, err
	} else {
		fd := 3 + len(*extraFiles)
		*extraFiles = append(*extraFiles, r)
		return fd, gob.NewEncoder(w), nil
	}
}

// Receive retrieves payload pipe fd from the environment,
// receives its payload and returns the Close method of the pipe.
func Receive(key string, e any, v **os.File) (func() error, error) {
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
			if v != nil {
				*v = setup
			}
		}
	}

	return setup.Close, gob.NewDecoder(setup).Decode(e)
}
