package helper

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
)

var (
	ErrContainsNull = errors.New("argument contains null character")
)

// Args is sealed with a slice of arguments for writing to the helper args FD.
// The sealing args is checked to not contain null characters.
// Attempting to seal an instance twice will cause a panic.
type Args interface {
	Seal(args []string) error
	io.WriterTo
	fmt.Stringer
}

// argsFD implements Args for helpers expecting null terminated arguments to a file descriptor.
// argsFD must not be copied after first use.
type argsFD struct {
	seal []byte
	sync.RWMutex
}

func (a *argsFD) Seal(args []string) error {
	a.Lock()
	defer a.Unlock()

	if a.seal != nil {
		panic("args sealed twice")
	}

	seal := bytes.Buffer{}

	n := 0
	for _, arg := range args {
		// reject argument strings containing null
		if hasNull(arg) {
			return ErrContainsNull
		}

		// accumulate buffer size
		n += len(arg) + 1
	}
	seal.Grow(n)

	// write null terminated arguments
	for _, arg := range args {
		seal.WriteString(arg)
		seal.WriteByte('\x00')
	}

	a.seal = seal.Bytes()
	return nil
}

func (a *argsFD) WriteTo(w io.Writer) (int64, error) {
	a.RLock()
	defer a.RUnlock()

	if a.seal == nil {
		panic("attempted to activate unsealed args")
	}

	n, err := w.Write(a.seal)
	return int64(n), err
}

func (a *argsFD) String() string {
	if a == nil {
		return "(invalid helper args)"
	}

	if a.seal == nil {
		return "(unsealed helper args)"
	}

	return strings.ReplaceAll(string(a.seal), "\x00", " ")
}

func hasNull(s string) bool {
	for _, b := range s {
		if b == '\x00' {
			return true
		}
	}
	return false
}

// NewArgs returns a new instance of Args
func NewArgs() Args {
	return new(argsFD)
}
