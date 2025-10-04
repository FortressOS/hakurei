// Package app implements high-level hakurei container behaviour.
package app

import (
	"context"
	"log"
	"os"

	"hakurei.app/container"
	"hakurei.app/hst"
	"hakurei.app/internal/app/state"
)

// Main runs an app according to [hst.Config] and terminates. Main does not return.
func Main(ctx context.Context, msg container.Msg, config *hst.Config) {
	var id state.ID
	if err := state.NewAppID(&id); err != nil {
		log.Fatal(err)
	}

	seal := outcome{syscallDispatcher: direct{}}
	if err := seal.finalise(ctx, msg, &id, config); err != nil {
		printMessageError("cannot seal app:", err)
		os.Exit(1)
	}

	seal.main(msg)
	panic("unreachable")
}
