package state_test

import (
	"testing"

	"hakurei.app/cmd/hakurei/internal/state"
)

func TestMulti(t *testing.T) { testStore(t, state.NewMulti(t.TempDir())) }
