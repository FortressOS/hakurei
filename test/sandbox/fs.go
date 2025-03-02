package sandbox

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"
)

var (
	ErrFSBadLength  = errors.New("bad dir length")
	ErrFSBadData    = errors.New("data differs")
	ErrFSBadMode    = errors.New("mode differs")
	ErrFSInvalidEnt = errors.New("invalid entry condition")
)

type FS struct {
	Mode fs.FileMode    `json:"mode"`
	Dir  map[string]*FS `json:"dir"`
	Data *string        `json:"data"`
}

func printDir(prefix string, dir []fs.DirEntry) {
	names := make([]string, len(dir))
	for i, ent := range dir {
		name := ent.Name()
		if ent.IsDir() {
			name += "/"
		}
		names[i] = fmt.Sprintf("%q", name)
	}
	printf("[FAIL] d %q: %s", prefix, strings.Join(names, " "))
}

func (s *FS) Compare(prefix string, e fs.FS) error {
	if s.Data != nil {
		if s.Dir != nil {
			panic("invalid state")
		}
		panic("invalid compare call")
	}

	if s.Dir == nil {
		printf("[ OK ] s %s", prefix)
		return nil
	}

	var dir []fs.DirEntry
	if d, err := fs.ReadDir(e, prefix); err != nil {
		return err
	} else if len(d) != len(s.Dir) {
		printDir(prefix, d)
		return ErrFSBadLength
	} else {
		dir = d
	}

	for _, got := range dir {
		name := got.Name()

		if want, ok := s.Dir[name]; !ok {
			printDir(prefix, dir)
			return fs.ErrNotExist
		} else if want.Dir != nil && !got.IsDir() {
			printDir(prefix, dir)
			return ErrFSInvalidEnt
		} else {
			name = path.Join(prefix, name)

			if fi, err := got.Info(); err != nil {
				return err
			} else if fi.Mode() != want.Mode {
				printf("[FAIL] m %q: %x, want %x",
					name, uint32(fi.Mode()), uint32(want.Mode))
				return ErrFSBadMode
			}

			if want.Data != nil {
				if want.Dir != nil {
					panic("invalid state")
				}
				if v, err := fs.ReadFile(e, name); err != nil {
					return err
				} else if string(v) != *want.Data {
					printf("[FAIL] f %s", name)
					return ErrFSBadData
				}
				printf("[ OK ] f %s", name)
			} else if err := want.Compare(name, e); err != nil {
				return err
			}
		}
	}
	printf("[ OK ] d %s", prefix)

	return nil
}
