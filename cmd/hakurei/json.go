package main

import (
	"encoding/json"
	"errors"
	"io"
	"strconv"
)

// decodeJSON decodes json from r and stores it in v. A non-nil error results in a call to fatal.
func decodeJSON(fatal func(v ...any), op string, r io.Reader, v any) {
	err := json.NewDecoder(r).Decode(v)
	if err == nil {
		return
	}

	var (
		syntaxError        *json.SyntaxError
		unmarshalTypeError *json.UnmarshalTypeError

		msg string
	)

	switch {
	case errors.As(err, &syntaxError) && syntaxError != nil:
		msg = syntaxError.Error() +
			" at byte " + strconv.FormatInt(syntaxError.Offset, 10)

	case errors.As(err, &unmarshalTypeError) && unmarshalTypeError != nil:
		msg = "inappropriate " + unmarshalTypeError.Value +
			" at byte " + strconv.FormatInt(unmarshalTypeError.Offset, 10)

	default:
		// InvalidUnmarshalError: incorrect usage, does not need to be handled
		// io.ErrUnexpectedEOF: no additional error information available
		msg = err.Error()
	}

	fatal("cannot " + op + ": " + msg)
}

// encodeJSON encodes v to output. A non-nil error results in a call to fatal.
func encodeJSON(fatal func(v ...any), output io.Writer, short bool, v any) {
	encoder := json.NewEncoder(output)
	if !short {
		encoder.SetIndent("", "  ")
	}

	if err := encoder.Encode(v); err != nil {
		var marshalerError *json.MarshalerError
		if errors.As(err, &marshalerError) && marshalerError != nil {
			// this likely indicates an implementation error in hst
			fatal("cannot encode json for " + marshalerError.Type.String() + ": " + marshalerError.Err.Error())
			return
		}

		// UnsupportedTypeError, UnsupportedValueError: incorrect usage, does not need to be handled
		fatal("cannot write json: " + err.Error())
	}
}
