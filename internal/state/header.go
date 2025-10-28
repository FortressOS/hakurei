package state

import (
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strconv"
	"syscall"

	"hakurei.app/hst"
)

const (
	// entryHeaderMagic are magic bytes at the beginning of the state entry file.
	entryHeaderMagic = "\x00\xff\xca\xfe"
	// entryHeaderRevision follows entryHeaderMagic and is incremented for revisions of the format.
	entryHeaderRevision = "\x00\x00"
	// entryHeaderSize is the fixed size of the header in bytes, including the enablement byte and its complement.
	entryHeaderSize = len(entryHeaderMagic+entryHeaderRevision) + 2
)

// entryHeaderEncode encodes a state entry header for a [hst.Enablement] byte.
func entryHeaderEncode(et hst.Enablement) *[entryHeaderSize]byte {
	data := [entryHeaderSize]byte([]byte(
		entryHeaderMagic + entryHeaderRevision + string([]hst.Enablement{et, ^et}),
	))
	return &data
}

// entryHeaderDecode validates a state entry header and returns the [hst.Enablement] byte.
func entryHeaderDecode(data *[entryHeaderSize]byte) (hst.Enablement, error) {
	if magic := data[:len(entryHeaderMagic)]; string(magic) != entryHeaderMagic {
		return 0, errors.New("invalid header " + hex.EncodeToString(magic))
	}
	if revision := data[len(entryHeaderMagic):len(entryHeaderMagic+entryHeaderRevision)]; string(revision) != entryHeaderRevision {
		return 0, errors.New("unexpected revision " + hex.EncodeToString(revision))
	}

	et := data[len(entryHeaderMagic+entryHeaderRevision)]
	if et != ^data[len(entryHeaderMagic+entryHeaderRevision)+1] {
		return 0, errors.New("header enablement value is inconsistent")
	}
	return hst.Enablement(et), nil
}

// EntrySizeError is returned for a file too small to hold a state entry header.
type EntrySizeError struct {
	Name string
	Size int64
}

func (e *EntrySizeError) Error() string {
	if e.Name == "" {
		return "state entry file is too short"
	}
	return "state entry file " + strconv.Quote(e.Name) + " is too short"
}

// entryCheckFile checks whether [os.FileInfo] refers to a file that might hold [hst.State].
func entryCheckFile(fi os.FileInfo) error {
	if fi.IsDir() {
		return syscall.EISDIR
	}
	if s := fi.Size(); s <= int64(entryHeaderSize) {
		return &EntrySizeError{Name: fi.Name(), Size: s}
	}
	return nil
}

// entryReadHeader reads [hst.Enablement] from an [io.Reader].
func entryReadHeader(r io.Reader) (hst.Enablement, error) {
	var data [entryHeaderSize]byte
	if n, err := r.Read(data[:]); err != nil {
		return 0, err
	} else if n != entryHeaderSize {
		return 0, &EntrySizeError{Size: int64(n)}
	}
	return entryHeaderDecode(&data)
}

// entryWriteHeader writes [hst.Enablement] header to an [io.Writer].
func entryWriteHeader(w io.Writer, et hst.Enablement) error {
	_, err := w.Write(entryHeaderEncode(et)[:])
	return err
}
