package command_test

import (
	"testing"

	"hakurei.app/command"
)

func TestBuild(t *testing.T) {
	c := command.New(nil, nil, "test", nil)
	stubHandler := func([]string) error { panic("unreachable") }

	t.Run("nil direct handler", func(t *testing.T) {
		defer checkRecover(t, "Command", "invalid handler")
		c.Command("name", "usage", nil)
	})

	t.Run("direct zero length", func(t *testing.T) {
		wantPanic := "invalid subcommand"
		t.Run("zero length name", func(t *testing.T) { defer checkRecover(t, "Command", wantPanic); c.Command("", "usage", stubHandler) })
		t.Run("zero length usage", func(t *testing.T) { defer checkRecover(t, "Command", wantPanic); c.Command("name", "", stubHandler) })
	})

	t.Run("direct adopt unique names", func(t *testing.T) {
		c.Command("d0", "usage", stubHandler)
		c.Command("d1", "usage", stubHandler)
	})

	t.Run("direct adopt non-unique name", func(t *testing.T) {
		defer checkRecover(t, "Command", "attempted to initialise subcommand with non-unique name")
		c.Command("d0", "usage", stubHandler)
	})

	t.Run("zero length", func(t *testing.T) {
		wantPanic := "invalid subcommand tree"
		t.Run("zero length name", func(t *testing.T) { defer checkRecover(t, "New", wantPanic); c.New("", "usage") })
		t.Run("zero length usage", func(t *testing.T) { defer checkRecover(t, "New", wantPanic); c.New("name", "") })
	})

	t.Run("direct adopt unique names", func(t *testing.T) {
		c.New("t0", "usage")
		c.New("t1", "usage")
	})

	t.Run("direct adopt non-unique name", func(t *testing.T) {
		defer checkRecover(t, "Command", "attempted to initialise subcommand tree with non-unique name")
		c.New("t0", "usage")
	})
}

func checkRecover(t *testing.T, name, wantPanic string) {
	if r := recover(); r != wantPanic {
		t.Errorf("%s: panic = %v; wantPanic %v",
			name, r, wantPanic)
	}
}
