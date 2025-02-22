// Package command implements generic nested command parsing.
package command

import (
	"flag"
	"strings"
)

type (
	// HandlerFunc is called when matching a directly handled subcommand tree.
	HandlerFunc = func(args []string) error

	// LogFunc is the function signature of a printf function.
	LogFunc = func(format string, a ...any)

	// FlagDefiner is a deferred flag definer value, usually encapsulating the default value.
	FlagDefiner interface {
		// Define defines the flag in set.
		Define(b *strings.Builder, set *flag.FlagSet, p any, name, usage string)
	}

	Flag[T any] interface {
		// Flag defines a generic flag type in Node's flag set.
		Flag(p any, name string, value FlagDefiner, usage string) T
	}

	Command interface {
		Parse(arguments []string) error
		baseNode[Command]
	}
	Node baseNode[Node]

	baseNode[T any] interface {
		// Command appends a subcommand with direct command handling.
		Command(name, usage string, f HandlerFunc) T

		// New returns a new subcommand tree.
		New(name, usage string) (sub Node)
		// NewCommand returns a new subcommand with direct command handling.
		NewCommand(name, usage string, f HandlerFunc) (sub Flag[Node])

		// PrintHelp prints a help message to the configured writer.
		PrintHelp()

		Flag[T]
	}
)
