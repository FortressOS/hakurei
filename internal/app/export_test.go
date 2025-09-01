package app

import (
	"context"

	"hakurei.app/container"
	"hakurei.app/internal/app/state"
	"hakurei.app/internal/sys"
	"hakurei.app/system"
)

func NewWithID(ctx context.Context, id state.ID, os sys.State) *App {
	return &App{id: newID(&id), sys: os, ctx: ctx}
}

func AppIParams(a *App, seal *Outcome) (*system.I, *container.Params) {
	if a.outcome != seal || a.id != seal.id {
		panic("broken app/outcome link")
	}
	return seal.sys, seal.container
}
