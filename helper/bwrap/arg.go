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

func (c *Config) FDArgs(syncFd *os.File, args *[]string, extraFiles *proc.ExtraFilesPre, files *[]proc.File) {
	builders := []FDBuilder{
		c.seccompArgs(),
		newFile(positionalArgs[SyncFd], syncFd),
	}

	argc := 0
	fc := 0
	for _, b := range builders {
		l := b.Len()
		if l < 1 {
			continue
		}
		argc += l
		fc++

		proc.InitFile(b, extraFiles)
	}

	fc++ // allocate extra slot for stat fd
	*args = slices.Grow(*args, argc)
	*files = slices.Grow(*files, fc)

	for _, b := range builders {
		if b.Len() < 1 {
			continue
		}

		b.Append(args)
		*files = append(*files, b)
	}
	return
}
