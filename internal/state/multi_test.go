package state_test

import (
	"testing"

	"git.gensokyo.uk/security/fortify/internal/state"
)

func TestMulti(t *testing.T) {
	testStore(t, state.NewMulti(t.TempDir()))
}
