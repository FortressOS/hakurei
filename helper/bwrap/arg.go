package bwrap

import (
	"os"
	"slices"

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
	proc.File
	Builder
}

// Args returns a slice of bwrap args corresponding to c.
func (c *Config) Args(syncFd *os.File, extraFiles *proc.ExtraFilesPre, files *[]proc.File) (args []string) {
	builders := []Builder{
		c.boolArgs(),
		c.intArgs(),
		c.stringArgs(),
		c.pairArgs(),
		c.seccompArgs(),
		newFile(positionalArgs[SyncFd], syncFd),
	}

	builders = slices.Grow(builders, len(c.Filesystem)+1)
	for _, f := range c.Filesystem {
		builders = append(builders, f)
	}
	builders = append(builders, c.Chmod)

	argc := 0
	fc := 0
	for _, b := range builders {
		l := b.Len()
		if l < 1 {
			continue
		}
		argc += l

		if f, ok := b.(FDBuilder); ok {
			fc++
			proc.InitFile(f, extraFiles)
		}
	}
	fc++ // allocate extra slot for stat fd

	args = make([]string, 0, argc)
	*files = slices.Grow(*files, fc)
	for _, b := range builders {
		if b.Len() < 1 {
			continue
		}
		b.Append(&args)

		if f, ok := b.(FDBuilder); ok {
			*files = append(*files, f)
		}
	}

	return
}
