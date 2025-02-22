package command

import (
	"flag"
	"fmt"
	"io"
)

// New initialises a root Node.
func New(output io.Writer, logf LogFunc, name string) Command {
	return rootNode{newNode(output, logf, name, "")}
}

func newNode(output io.Writer, logf LogFunc, name, usage string) *node {
	n := &node{
		name: name, usage: usage,
		out: output, logf: logf,
		set: flag.NewFlagSet(name, flag.ContinueOnError),
	}
	n.set.SetOutput(output)
	n.set.Usage = func() {
		_ = n.writeHelp()
		if n.suffix.Len() > 0 {
			_, _ = fmt.Fprintln(output, "Flags:")
			n.set.PrintDefaults()
			_, _ = fmt.Fprintln(output)
		}
	}

	return n
}

func (n *node) Command(name, usage string, f HandlerFunc) Node {
	n.NewCommand(name, usage, f)
	return n
}

func (n *node) NewCommand(name, usage string, f HandlerFunc) Flag[Node] {
	if f == nil {
		panic("invalid handler")
	}
	if name == "" || usage == "" {
		panic("invalid subcommand")
	}

	s := newNode(n.out, n.logf, name, usage)
	s.f = f
	if !n.adopt(s) {
		panic("attempted to initialise subcommand with non-unique name")
	}
	return s
}

func (n *node) New(name, usage string) Node {
	if name == "" || usage == "" {
		panic("invalid subcommand tree")
	}
	s := newNode(n.out, n.logf, name, usage)
	if !n.adopt(s) {
		panic("attempted to initialise subcommand tree with non-unique name")
	}
	return s
}
