package state_test

import (
	"testing"

	"git.ophivana.moe/security/fortify/internal/state"
)

func TestMulti(t *testing.T) {
	testStore(t, state.NewMulti(t.TempDir()))
}
