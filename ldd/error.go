package ldd

import "fmt"

type EntryUnexpectedSegmentsError struct {
	Entry string
}

func (e *EntryUnexpectedSegmentsError) Error() string {
	return fmt.Sprintf("unexpected segments in entry %q", e.Entry)
}
