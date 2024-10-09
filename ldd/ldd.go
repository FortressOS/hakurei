package ldd

import (
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
)

var (
	ErrUnexpectedSeparator = errors.New("unexpected separator")
	ErrPathNotAbsolute     = errors.New("path not absolute")
	ErrBadLocationFormat   = errors.New("bad location format")
)

type Entry struct {
	Name     string `json:"name,omitempty"`
	Path     string `json:"path,omitempty"`
	Location uint64 `json:"location"`
}

func Exec(p string) ([]*Entry, error) {
	t := exec.Command("ldd", p)
	t.Stdout = new(strings.Builder)
	t.Stderr = os.Stderr

	if err := t.Run(); err != nil {
		return nil, err
	}

	out := t.Stdout.(fmt.Stringer).String()
	payload := strings.Split(out, "\n")

	result := make([]*Entry, len(payload))

	for i, ent := range payload {
		if len(ent) == 0 {
			continue
		}

		segment := strings.SplitN(ent, " ", 5)

		var iL int

		switch len(segment) {
		case 2: // /lib/ld-musl-x86_64.so.1 (0x7f04d14ef000)
			iL = 1
			result[i] = &Entry{Name: segment[0]}
		case 4: // libc.musl-x86_64.so.1 => /lib/ld-musl-x86_64.so.1 (0x7f04d14ef000)
			iL = 3
			if segment[1] != "=>" {
				return nil, ErrUnexpectedSeparator
			}
			if !path.IsAbs(segment[2]) {
				return nil, ErrPathNotAbsolute
			}
			result[i] = &Entry{
				Name: segment[0],
				Path: segment[2],
			}
		default:
			return nil, &EntryUnexpectedSegmentsError{ent}
		}

		if loc, err := parseLocation(segment[iL]); err != nil {
			return nil, err
		} else {
			result[i].Location = loc
		}
	}

	return result, nil
}

func parseLocation(s string) (uint64, error) {
	if len(s) < 4 || s[len(s)-1] != ')' || s[:3] != "(0x" {
		return math.MaxUint64, ErrBadLocationFormat
	}
	return strconv.ParseUint(s[3:len(s)-1], 16, 64)
}
