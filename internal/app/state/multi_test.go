package state_test

import (
	"testing"

	"hakurei.app/internal/app/state"
)

func TestMulti(t *testing.T) { testStore(t, state.NewMulti(t.TempDir())) }
