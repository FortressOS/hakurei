package bwrap

import (
	"encoding/gob"
	"os"
	"slices"
	"strconv"

	"git.gensokyo.uk/security/fortify/helper/proc"
)

type Builder interface {
	Len() int
	Append(args *[]string)
}

type FSBuilder interface {
	Path() string
	Builder
}

type FDBuilder interface {
	Len() int
	Append(args *[]string, extraFiles *[]*os.File) error
}

func init() {
	gob.Register(new(pairF))
	gob.Register(new(stringF))
}

type pairF [3]string

func (p *pairF) Path() string {
	return p[2]
}

func (p *pairF) Len() int {
	return len(p) // compiler replaces this with 3
}

func (p *pairF) Append(args *[]string) {
	*args = append(*args, p[0], p[1], p[2])
}

type stringF [2]string

func (s stringF) Path() string {
	return s[1]
}

func (s stringF) Len() int {
	return len(s) // compiler replaces this with 2
}

func (s stringF) Append(args *[]string) {
	*args = append(*args, s[0], s[1])
}

type fileF struct {
	name string
	file *os.File
}

func (f *fileF) Len() int {
	if f.file == nil {
		return 0
	}
	return 2
}

func (f *fileF) Append(args *[]string, extraFiles *[]*os.File) error {
	if f.file == nil {
		return nil
	}
	extraFile(args, extraFiles, f.name, f.file)
	return nil
}

func extraFile(args *[]string, extraFiles *[]*os.File, name string, f *os.File) {
	if f == nil {
		return
	}
	*args = append(*args, name, strconv.Itoa(int(proc.ExtraFileSlice(extraFiles, f))))
}

// Args returns a slice of bwrap args corresponding to c.
func (c *Config) Args() (args []string) {
	builders := []Builder{
		c.boolArgs(),
		c.intArgs(),
		c.stringArgs(),
		c.pairArgs(),
	}

	// copy FSBuilder slice to builder slice
	fb := make([]Builder, len(c.Filesystem)+1)
	for i, f := range c.Filesystem {
		fb[i] = f
	}
	fb[len(fb)-1] = c.Chmod
	builders = append(builders, fb...)

	// accumulate arg count
	argc := 0
	for _, b := range builders {
		argc += b.Len()
	}

	args = make([]string, 0, argc)
	for _, b := range builders {
		b.Append(&args)
	}

	return
}

func (c *Config) FDArgs(syncFd *os.File, extraFiles *[]*os.File) (args []string, err error) {
	builders := []FDBuilder{
		&seccompBuilder{c},
		&fileF{positionalArgs[SyncFd], syncFd},
	}

	argc := 0
	for _, b := range builders {
		argc += b.Len()
	}

	args = make([]string, 0, argc)
	*extraFiles = slices.Grow(*extraFiles, len(builders))

	for _, b := range builders {
		if err = b.Append(&args, extraFiles); err != nil {
			break
		}
	}
	return
}
