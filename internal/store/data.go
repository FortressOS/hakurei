package store

import (
	"encoding/gob"
	"fmt"
	"io"
	"os"

	"hakurei.app/hst"
)

// entryEncode encodes [hst.State] into [io.Writer] with the state entry header.
// entryEncode does not validate the embedded [hst.Config] value.
//
// A non-nil error returned by entryEncode is of type [hst.AppError].
func entryEncode(w io.Writer, s *hst.State) error {
	if err := entryWriteHeader(w, s.Enablements.Unwrap()); err != nil {
		return &hst.AppError{Step: "encode state header", Err: err}
	} else if err = gob.NewEncoder(w).Encode(s); err != nil {
		return &hst.AppError{Step: "encode state body", Err: err}
	} else {
		return nil
	}
}

// entryDecodeHeader calls entryReadHeader, returning [hst.AppError] for a non-nil error.
func entryDecodeHeader(r io.Reader) (hst.Enablement, error) {
	if et, err := entryReadHeader(r); err != nil {
		return 0, &hst.AppError{Step: "decode state header", Err: err}
	} else {
		return et, nil
	}
}

// entryDecode decodes [hst.State] from [io.Reader] and stores the result in the value pointed to by p.
// entryDecode validates the embedded [hst.Config] value.
//
// A non-nil error returned by entryDecode is of type [hst.AppError].
func entryDecode(r io.Reader, p *hst.State) (hst.Enablement, error) {
	if et, err := entryDecodeHeader(r); err != nil {
		return et, err
	} else if err = gob.NewDecoder(r).Decode(&p); err != nil {
		return et, &hst.AppError{Step: "decode state body", Err: err}
	} else if err = p.Config.Validate(); err != nil {
		return et, err
	} else if p.Enablements.Unwrap() != et {
		return et, &hst.AppError{Step: "validate state enablement", Err: os.ErrInvalid,
			Msg: fmt.Sprintf("state entry %s has unexpected enablement byte %#x, %#x", p.ID.String(), byte(p.Enablements.Unwrap()), byte(et))}
	} else {
		return et, nil
	}
}
