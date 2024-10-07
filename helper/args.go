package helper

import (
	"errors"
	"io"
	"strings"
)

var (
	ErrContainsNull = errors.New("argument contains null character")
)

type argsWt []string

// checks whether any element contains the null character
// must be called before args use and args must not be modified after call
func (a argsWt) check() error {
	for _, arg := range a {
		for _, b := range arg {
			if b == '\x00' {
				return ErrContainsNull
			}
		}
	}

	return nil
}

func (a argsWt) WriteTo(w io.Writer) (int64, error) {
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

func (a argsWt) String() string {
	return strings.Join(a, " ")
}

// NewCheckedArgs returns a checked argument writer for args.
// Callers must not retain any references to args.
func NewCheckedArgs(args []string) (io.WriterTo, error) {
	a := argsWt(args)
	return a, a.check()
}

// MustNewCheckedArgs returns a checked argument writer for args and panics if check fails.
// Callers must not retain any references to args.
func MustNewCheckedArgs(args []string) io.WriterTo {
	a, err := NewCheckedArgs(args)
	if err != nil {
		panic(err.Error())
	}

	return a
}
