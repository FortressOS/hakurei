package ldd

import (
	"errors"
	"fmt"
)

var (
	ErrUnexpectedSeparator = errors.New("unexpected separator")
	ErrPathNotAbsolute     = errors.New("path not absolute")
	ErrBadLocationFormat   = errors.New("bad location format")
	ErrUnexpectedNewline   = errors.New("unexpected newline")
)

type EntryUnexpectedSegmentsError string

func (e EntryUnexpectedSegmentsError) Is(err error) bool {
	var eq EntryUnexpectedSegmentsError
	if !errors.As(err, &eq) {
		return false
	}
	return e == eq
}

func (e EntryUnexpectedSegmentsError) Error() string {
	return fmt.Sprintf("unexpected segments in entry %q", string(e))
}
