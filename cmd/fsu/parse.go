package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"
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
			return -1, fmt.Errorf("invalid character '%s' at index %d", string([]byte{ch}), i)
		}
		n = n*10 + int(ch)
	}
	return n, nil
}

func parseConfig(p string, puid int) (fid int, ok bool) {
	// refuse to run if fsurc is not protected correctly
	if s, err := os.Stat(p); err != nil {
		log.Fatal(err)
	} else if s.Mode().Perm() != 0400 {
		log.Fatal("bad fsurc perm")
	} else if st := s.Sys().(*syscall.Stat_t); st.Uid != 0 || st.Gid != 0 {
		log.Fatal("fsurc must be owned by uid 0")
	}

	if r, err := os.Open(p); err != nil {
		log.Fatal(err)
		return -1, false
	} else {
		s := bufio.NewScanner(r)
		var line int
		for s.Scan() {
			line++

			// <puid> <fid>
			lf := strings.SplitN(s.Text(), " ", 2)
			if len(lf) != 2 {
				log.Fatalf("invalid entry on line %d", line)
			}

			var puid0 int
			if puid0, err = parseUint32Fast(lf[0]); err != nil || puid0 < 1 {
				log.Fatalf("invalid parent uid on line %d", line)
			}

			ok = puid0 == puid
			if ok {
				// allowed fid range 0 to 99
				if fid, err = parseUint32Fast(lf[1]); err != nil || fid < 0 || fid > 99 {
					log.Fatalf("invalid fortify uid on line %d", line)
				}
				return
			}
		}
		if err = s.Err(); err != nil {
			log.Fatalf("cannot read fsurc: %v", err)
		}
		return -1, false
	}
}
