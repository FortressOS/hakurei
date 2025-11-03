package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strings"
)

const (
	// useridStart is the first userid.
	useridStart = 0
	// useridEnd is the last userid.
	useridEnd = useridStart + rangeSize - 1
)

// parseUint32Fast parses a string representation of an unsigned 32-bit integer value
// using the fast path only. This limits the range of values it is defined in.
func parseUint32Fast(s string) (uint32, error) {
	sLen := len(s)
	if sLen < 1 {
		return 0, errors.New("zero length string")
	}
	if sLen > 10 {
		return 0, errors.New("string too long")
	}

	var n uint32
	for i, ch := range []byte(s) {
		ch -= '0'
		if ch > 9 {
			return 0, fmt.Errorf("invalid character '%s' at index %d", string(ch+'0'), i)
		}
		n = n*10 + uint32(ch)
	}
	return n, nil
}

// parseConfig reads a list of allowed users from r until it encounters puid or [io.EOF].
//
// Each line of the file specifies a hakurei userid to kernel uid mapping. A line consists
// of the string representation of the uid of the user wishing to start hakurei containers,
// followed by a space, followed by the string representation of its userid. Duplicate uid
// entries are ignored, with the first occurrence taking effect.
//
// All string representations are parsed by calling parseUint32Fast.
func parseConfig(r io.Reader, puid uint32) (userid uint32, ok bool, err error) {
	s := bufio.NewScanner(r)
	var (
		line  uintptr
		puid0 uint32
	)
	for s.Scan() {
		line++

		// <puid> <userid>
		lf := strings.SplitN(s.Text(), " ", 2)
		if len(lf) != 2 {
			return useridEnd + 1, false, fmt.Errorf("invalid entry on line %d", line)
		}

		puid0, err = parseUint32Fast(lf[0])
		if err != nil || puid0 < 1 {
			return useridEnd + 1, false, fmt.Errorf("invalid parent uid on line %d", line)
		}

		ok = puid0 == puid
		if ok {
			// userid bound to a range, uint32 size allows this to be increased if needed
			if userid, err = parseUint32Fast(lf[1]); err != nil ||
				userid < useridStart || userid > useridEnd {
				return useridEnd + 1, false, fmt.Errorf("invalid userid on line %d", line)
			}
			return
		}
	}
	return useridEnd + 1, false, s.Err()
}

// hsuConfPath is an absolute pathname to the hsu configuration file.
// Its contents are interpreted by parseConfig.
const hsuConfPath = "/etc/hsurc"

// mustParseConfig calls parseConfig to interpret the contents of hsuConfPath,
// terminating the program if an error is encountered, the syntax is incorrect,
// or the current user is not authorised to use hsu because its uid is missing.
//
// Therefore, code after this function call can assume an authenticated state.
//
// mustParseConfig returns the userid value of the current user.
func mustParseConfig(puid int) (userid uint32) {
	if puid > math.MaxUint32 {
		log.Fatalf("got impossible uid %d", puid)
	}

	var ok bool
	if f, err := os.Open(hsuConfPath); err != nil {
		log.Fatal(err)
	} else if userid, ok, err = parseConfig(f, uint32(puid)); err != nil {
		log.Fatal(err)
	} else if err = f.Close(); err != nil {
		log.Fatal(err)
	}
	if !ok {
		log.Fatalf("uid %d is not in the hsurc file", puid)
	}

	return
}

// envIdentity is the name of the environment variable holding a
// string representation of the current application identity.
var envIdentity = "HAKUREI_IDENTITY"

// mustReadIdentity calls parseUint32Fast to interpret the value stored in envIdentity,
// terminating the program if the value is not set, malformed, or out of bounds.
func mustReadIdentity() uint32 {
	// ranges defined in hst and copied to this package to avoid importing hst
	if as, ok := os.LookupEnv(envIdentity); !ok {
		log.Fatal("HAKUREI_IDENTITY not set")
		panic("unreachable")
	} else if identity, err := parseUint32Fast(as); err != nil ||
		identity < identityStart || identity > identityEnd {
		log.Fatal("invalid identity")
		panic("unreachable")
	} else {
		return identity
	}
}
