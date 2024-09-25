package helper

import (
	"errors"
	"io"
	"strings"
)

var (
	ErrContainsNull = errors.New("argument contains null character")
)

type argsFD []string

// checks whether any element contains the null character
// must be called before args use and args must not be modified after call
func (a argsFD) check() error {
	for _, arg := range a {
		for _, b := range arg {
			if b == '\x00' {
				return ErrContainsNull
			}
		}
	}

	return nil
}

func (a argsFD) WriteTo(w io.Writer) (int64, error) {
	// assuming already checked

	nt := 0
	// write null terminated arguments
	for _, arg := range a {
		n, err := w.Write([]byte(arg + "\x00"))
		nt += n

		if err != nil {
			return int64(nt), err
		}
	}

	return int64(nt), nil
}

func (a argsFD) String() string {
	if a == nil {
		return "(invalid helper args)"
	}

	return strings.Join(a, " ")
}

// NewCheckedArgs returns a checked argument writer for args.
// Callers must not retain any references to args.
func NewCheckedArgs(args []string) (io.WriterTo, error) {
	a := argsFD(args)
	return a, a.check()
}
