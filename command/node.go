package command

import (
	"flag"
	"io"
)

type node struct {
	child, next *node
	name, usage string

	out  io.Writer
	logf LogFunc

	f   HandlerFunc
	set *flag.FlagSet
}

func (n *node) adopt(v *node) bool {
	if n.child != nil {
		return n.child.append(v)
	}
	n.child = v
	return true
}

func (n *node) append(v *node) bool {
	if n.name == v.name {
		return false
	}
	if n.next != nil {
		return n.next.append(v)
	}
	n.next = v
	return true
}
