package command

import (
	"errors"
	"log"
)

var (
	ErrEmptyTree = errors.New("subcommand tree has no nodes")
	ErrNoMatch   = errors.New("did not match any subcommand")
)

func (n *node) Parse(arguments []string) error {
	if n.usage == "" { // root node has zero length usage
		if n.next != nil {
			panic("invalid toplevel state")
		}
		goto match
	}

	if len(arguments) == 0 {
		// unreachable: zero length args cause upper level to return with a help message
		panic("attempted to parse with zero length args")
	}
	if arguments[0] != n.name {
		if n.next == nil {
			n.printf("%q is not a valid command", arguments[0])
			return ErrNoMatch
		}
		n.next.prefix = n.prefix
		return n.next.Parse(arguments)
	}
	arguments = arguments[1:]

match:
	if n.child != nil {
		// propagate help prefix early: flag set usage dereferences help
		n.child.prefix = append(n.prefix, n.name)
	}

	if n.set.Parsed() {
		panic("invalid set state")
	}
	if err := n.set.Parse(arguments); err != nil {
		return FlagError{err}
	}
	args := n.set.Args()

	if n.child != nil {
		if n.f != nil {
			if n.usage != "" { // root node early special case
				panic("invalid subcommand tree state")
			}

			// special case: root node calls HandlerFunc for initialisation
			if err := n.f(nil); err != nil {
				return err
			}
		}

		if len(args) == 0 {
			return n.writeHelp()
		}
		return n.child.Parse(args)
	}

	if n.f == nil {
		n.printf("%q has no subcommands", n.name)
		return ErrEmptyTree
	}
	return n.f(args)
}

func (n *node) printf(format string, a ...any) {
	if n.logf == nil {
		log.Printf(format, a...)
	} else {
		n.logf(format, a...)
	}
}
