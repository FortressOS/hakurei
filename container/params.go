package container

import (
	"encoding/gob"
	"errors"
	"os"
	"strconv"
	"syscall"
)

var (
	ErrNotSet = errors.New("environment variable not set")
)

// Setup appends the read end of a pipe for setup params transmission and returns its fd.
func Setup(extraFiles *[]*os.File) (int, *gob.Encoder, error) {
	if r, w, err := os.Pipe(); err != nil {
		return -1, nil, err
	} else {
		fd := 3 + len(*extraFiles)
		*extraFiles = append(*extraFiles, r)
		return fd, gob.NewEncoder(w), nil
	}
}

// Receive retrieves setup fd from the environment and receives params.
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
				return nil, syscall.EBADF
			}
			if v != nil {
				*v = setup
			}
		}
	}

	return setup.Close, gob.NewDecoder(setup).Decode(e)
}
