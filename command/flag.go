package command

import (
	"errors"
	"flag"
	"strings"
)

// FlagError wraps errors returned by [flag].
type FlagError struct{ error }

func (e FlagError) Success() bool { return errors.Is(e.error, flag.ErrHelp) }
func (e FlagError) Is(target error) bool {
	return (e.error == nil && target == nil) ||
		((e.error != nil && target != nil) && e.error.Error() == target.Error())
}

func (n *node) Flag(p any, name string, value FlagDefiner, usage string) Node {
	value.Define(&n.suffix, n.set, p, name, usage)
	return n
}

// StringFlag is the default value of a string flag.
type StringFlag string

func (v StringFlag) Define(b *strings.Builder, set *flag.FlagSet, p any, name, usage string) {
	set.StringVar(p.(*string), name, string(v), usage)
	b.WriteString(" [" + prettyFlag(name) + " <value>]")
}

// BoolFlag is the default value of a bool flag.
type BoolFlag bool

func (v BoolFlag) Define(b *strings.Builder, set *flag.FlagSet, p any, name, usage string) {
	set.BoolVar(p.(*bool), name, bool(v), usage)
	b.WriteString(" [" + prettyFlag(name) + "]")
}

// this has no effect on parse outcome
func prettyFlag(name string) string {
	switch len(name) {
	case 0:
		panic("zero length flag name")
	case 1:
		return "-" + name
	default:
		return "--" + name
	}
}
