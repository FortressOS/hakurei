package helper

import (
	"bytes"
	"io"
	"syscall"
)

type argsWt [][]byte

func (a argsWt) WriteTo(w io.Writer) (int64, error) {
	nt := 0
	for _, arg := range a {
		n, err := w.Write(arg)
		nt += n

		if err != nil {
			return int64(nt), err
		}
	}

	return int64(nt), nil
}

func (a argsWt) String() string {
	return string(
		bytes.TrimSuffix(
			bytes.ReplaceAll(
				bytes.Join(a, nil),
				[]byte{0}, []byte{' '},
			),
			[]byte{' '},
		),
	)
}

// NewCheckedArgs returns a checked null-terminated argument writer for a copy of args.
func NewCheckedArgs(args ...string) (wt io.WriterTo, err error) {
	a := make(argsWt, len(args))
	for i, arg := range args {
		a[i], err = syscall.ByteSliceFromString(arg)
		if err != nil {
			return
		}
	}
	wt = a
	return
}

// MustNewCheckedArgs returns a checked null-terminated argument writer for a copy of args.
// If s contains a NUL byte this function panics instead of returning an error.
func MustNewCheckedArgs(args ...string) io.WriterTo {
	a, err := NewCheckedArgs(args...)
	if err != nil {
		panic(err.Error())
	}

	return a
}
