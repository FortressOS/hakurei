// Package app implements high-level hakurei container behaviour.
package app

import (
	"context"
	"log"
	"os"

	"hakurei.app/hst"
	"hakurei.app/internal/app/state"
	"hakurei.app/internal/sys"
)

// Main runs an app according to [hst.Config] and terminates. Main does not return.
func Main(ctx context.Context, k sys.State, config *hst.Config) {
	var id state.ID
	if err := state.NewAppID(&id); err != nil {
		log.Fatal(err)
	}

	var seal outcome
	seal.id = &stringPair[state.ID]{id, id.String()}
	if err := seal.finalise(ctx, k, config); err != nil {
		printMessageError("cannot seal app:", err)
		os.Exit(1)
	}

	seal.main()
	panic("unreachable")
}
