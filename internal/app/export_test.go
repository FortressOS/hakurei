package app

import (
	"context"

	"hakurei.app/container"
	"hakurei.app/hst"
	"hakurei.app/internal/app/state"
	"hakurei.app/internal/sys"
	"hakurei.app/system"
)

func FinaliseIParams(ctx context.Context, k sys.State, config *hst.Config, id *state.ID) (*system.I, *container.Params, error) {
	seal := outcome{id: &stringPair[state.ID]{*id, id.String()}}
	return seal.sys, seal.container, seal.finalise(ctx, k, config)
}
