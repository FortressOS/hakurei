package command

import (
	"flag"
	"testing"
)

func TestParseUnreachable(t *testing.T) {
	// top level bypasses name matching and recursive calls to Parse
	// returns when encountering zero-length args
	t.Run("zero-length args", func(t *testing.T) {
		defer checkRecover(t, "Parse", "attempted to parse with zero length args")
		_ = newNode(panicWriter{}, nil, " ", " ").Parse(nil)
	})

	// top level must not have siblings
	t.Run("toplevel siblings", func(t *testing.T) {
		defer checkRecover(t, "Parse", "invalid toplevel state")
		n := newNode(panicWriter{}, nil, " ", "")
		n.append(newNode(panicWriter{}, nil, "  ", " "))
		_ = n.Parse(nil)
	})

	// a node with descendents must not have a direct handler
	t.Run("sub handle conflict", func(t *testing.T) {
		defer checkRecover(t, "Parse", "invalid subcommand tree state")
		n := newNode(panicWriter{}, nil, " ", " ")
		n.adopt(newNode(panicWriter{}, nil, " ", " "))
		n.f = func([]string) error { panic("unreachable") }
		_ = n.Parse([]string{" "})
	})

	// this would only happen if a node was matched twice
	t.Run("parsed flag set", func(t *testing.T) {
		defer checkRecover(t, "Parse", "invalid set state")
		n := newNode(panicWriter{}, nil, " ", "")
		set := flag.NewFlagSet("parsed", flag.ContinueOnError)
		set.SetOutput(panicWriter{})
		_ = set.Parse(nil)
		n.set = set
		_ = n.Parse(nil)
	})
}

type panicWriter struct{}

func (p panicWriter) Write([]byte) (int, error) { panic("unreachable") }

func checkRecover(t *testing.T, name, wantPanic string) {
	if r := recover(); r != wantPanic {
		t.Errorf("%s: panic = %v; wantPanic %v",
			name, r, wantPanic)
	}
}
