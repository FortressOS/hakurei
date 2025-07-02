package state_test

import (
	"testing"

	"git.gensokyo.uk/security/hakurei/cmd/hakurei/internal/state"
)

func TestMulti(t *testing.T) { testStore(t, state.NewMulti(t.TempDir())) }
