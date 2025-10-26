package state_test

import (
	"log"
	"testing"

	"hakurei.app/container/check"
	"hakurei.app/internal/app/state"
	"hakurei.app/message"
)

func TestMulti(t *testing.T) {
	testStore(t, state.NewMulti(message.NewMsg(log.New(log.Writer(), "multi: ", 0)), check.MustAbs(t.TempDir())))
}
