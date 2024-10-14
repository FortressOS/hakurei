package bwrap

import "encoding/gob"

type Builder interface {
	Len() int
	Append(args *[]string)
}

type FSBuilder interface {
	Path() string
	Builder
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
