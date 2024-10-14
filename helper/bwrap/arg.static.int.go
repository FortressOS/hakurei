package bwrap

import "strconv"

const (
	UID = iota
	GID
	Perms
	Size
)

var intArgs = [...]string{
	UID:   "--uid",
	GID:   "--gid",
	Perms: "--perms",
	Size:  "--size",
}

func (c *Config) intArgs() Builder {
	// Arg types:
	//   Perms
	// are handled by the sequential builder

	return &intArg{
		UID: c.UID,
		GID: c.GID,
	}
}

type intArg [len(intArgs)]*int

func (n *intArg) Len() (l int) {
	for _, v := range n {
		if v != nil {
			l += 2
		}
	}
	return
}

func (n *intArg) Append(args *[]string) {
	for i, v := range n {
		if v != nil {
			*args = append(*args, intArgs[i], strconv.Itoa(*v))
		}
	}
}
