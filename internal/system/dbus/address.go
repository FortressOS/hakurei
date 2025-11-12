package dbus

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
)

type AddrEntry struct {
	Method string      `json:"method"`
	Values [][2]string `json:"values"`
}

// EqualAddrEntries returns whether two slices of [AddrEntry] are equal.
func EqualAddrEntries(entries, target []AddrEntry) bool {
	return slices.EqualFunc(entries, target, func(a AddrEntry, b AddrEntry) bool {
		return a.Method == b.Method && slices.Equal(a.Values, b.Values)
	})
}

// Parse parses D-Bus address according to
// https://dbus.freedesktop.org/doc/dbus-specification.html#addresses
func Parse(addr []byte) ([]AddrEntry, error) {
	// Look for a semicolon
	address := bytes.Split(bytes.TrimSuffix(addr, []byte{';'}), []byte{';'})

	// Allocate for entries
	v := make([]AddrEntry, len(address))

	for i, s := range address {
		var pairs [][]byte

		// Look for the colon :
		if method, list, ok := bytes.Cut(s, []byte{':'}); !ok {
			return v, &BadAddressError{ErrNoColon, i, s, -1, nil}
		} else {
			pairs = bytes.Split(list, []byte{','})
			v[i].Method = string(method)
			v[i].Values = make([][2]string, len(pairs))
		}

		for j, pair := range pairs {
			key, value, ok := bytes.Cut(pair, []byte{'='})
			if !ok {
				return v, &BadAddressError{ErrBadPairSep, i, s, j, pair}
			}
			if len(key) == 0 {
				return v, &BadAddressError{ErrBadPairKey, i, s, j, pair}
			}
			if len(value) == 0 {
				return v, &BadAddressError{ErrBadPairVal, i, s, j, pair}
			}
			v[i].Values[j][0] = string(key)

			if val, errno := unescapeValue(value); errno != errSuccess {
				return v, &BadAddressError{errno, i, s, j, pair}
			} else {
				v[i].Values[j][1] = string(val)
			}
		}
	}

	return v, nil
}

func unescapeValue(v []byte) (val []byte, errno ParseError) {
	if l := len(v) - (bytes.Count(v, []byte{'%'}) * 2); l < 0 {
		errno = ErrBadValLength
		return
	} else {
		val = make([]byte, l)
	}

	var i, skip int
	for iu, b := range v {
		if skip > 0 {
			skip--
			continue
		}

		if ib := bytes.IndexByte([]byte("-_/.\\*"), b); ib != -1 { // - // _/.\*
			goto opt
		} else if b >= '0' && b <= '9' { // 0-9
			goto opt
		} else if b >= 'A' && b <= 'Z' { // A-Z
			goto opt
		} else if b >= 'a' && b <= 'z' { // a-z
			goto opt
		}

		if b != '%' {
			errno = ErrBadValByte
			break
		}

		skip += 2
		if iu+2 >= len(v) {
			errno = ErrBadValHexLength
			break
		}
		if c, err := hex.Decode(val[i:i+1], v[iu+1:iu+3]); err != nil {
			if errors.As(err, new(hex.InvalidByteError)) {
				errno = ErrBadValHexByte
				break
			}
			// unreachable
			panic(err.Error())
		} else if c != 1 {
			// unreachable
			panic(fmt.Sprintf("invalid decode length %d", c))
		}
		i++
		continue

	opt:
		val[i] = b
		i++
	}

	return
}

type ParseError uint8

func (e ParseError) Error() string {
	switch e {
	case errSuccess:
		panic("attempted to return success as error")
	case ErrNoColon:
		return "address does not contain a colon"
	case ErrBadPairSep:
		return "'=' character not found"
	case ErrBadPairKey:
		return "'=' character has no key preceding it"
	case ErrBadPairVal:
		return "'=' character has no value following it"
	case ErrBadValLength:
		return "unescaped value has impossible length"
	case ErrBadValByte:
		return "in D-Bus address, characters other than [-0-9A-Za-z_/.\\*] should have been escaped"
	case ErrBadValHexLength:
		return "in D-Bus address, percent character was not followed by two hex digits"
	case ErrBadValHexByte:
		return "in D-Bus address, percent character was followed by characters other than hex digits"

	default:
		return fmt.Sprintf("parse error %d", e)
	}
}

const (
	errSuccess ParseError = iota
	ErrNoColon
	ErrBadPairSep
	ErrBadPairKey
	ErrBadPairVal
	ErrBadValLength
	ErrBadValByte
	ErrBadValHexLength
	ErrBadValHexByte
)

type BadAddressError struct {
	// error type
	Type ParseError

	// bad entry position
	EntryPos int
	// bad entry value
	EntryVal []byte

	// bad pair position
	PairPos int
	// bad pair value
	PairVal []byte
}

func (a *BadAddressError) Is(err error) bool {
	var b *BadAddressError
	return errors.As(err, &b) && a.Type == b.Type &&
		a.EntryPos == b.EntryPos && slices.Equal(a.EntryVal, b.EntryVal) &&
		a.PairPos == b.PairPos && slices.Equal(a.PairVal, b.PairVal)
}

func (a *BadAddressError) Error() string {
	return a.Type.Error()
}

func (a *BadAddressError) Unwrap() error {
	return a.Type
}
