package state_test

import (
	"log"
	"testing"

	"hakurei.app/container"
	"hakurei.app/internal/app/state"
)

func TestMulti(t *testing.T) {
	testStore(t, state.NewMulti(container.NewMsg(log.New(log.Writer(), "multi: ", 0)), t.TempDir()))
}
