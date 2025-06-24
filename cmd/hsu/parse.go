package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
)

func parseUint32Fast(s string) (int, error) {
	sLen := len(s)
	if sLen < 1 {
		return -1, errors.New("zero length string")
	}
	if sLen > 10 {
		return -1, errors.New("string too long")
	}

	n := 0
	for i, ch := range []byte(s) {
		ch -= '0'
		if ch > 9 {
			return -1, fmt.Errorf("invalid character '%s' at index %d", string(ch+'0'), i)
		}
		n = n*10 + int(ch)
	}
	return n, nil
}

func parseConfig(r io.Reader, puid int) (fid int, ok bool, err error) {
	s := bufio.NewScanner(r)
	var line, puid0 int
	for s.Scan() {
		line++

		// <puid> <fid>
		lf := strings.SplitN(s.Text(), " ", 2)
		if len(lf) != 2 {
			return -1, false, fmt.Errorf("invalid entry on line %d", line)
		}

		puid0, err = parseUint32Fast(lf[0])
		if err != nil || puid0 < 1 {
			return -1, false, fmt.Errorf("invalid parent uid on line %d", line)
		}

		ok = puid0 == puid
		if ok {
			// allowed fid range 0 to 99
			if fid, err = parseUint32Fast(lf[1]); err != nil || fid < 0 || fid > 99 {
				return -1, false, fmt.Errorf("invalid identity on line %d", line)
			}
			return
		}
	}
	return -1, false, s.Err()
}

func mustParseConfig(r io.Reader, puid int) (int, bool) {
	fid, ok, err := parseConfig(r, puid)
	if err != nil {
		log.Fatal(err)
	}
	return fid, ok
}
