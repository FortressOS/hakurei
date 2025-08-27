package app

import (
	"hakurei.app/container"
	"hakurei.app/internal/app/state"
	"hakurei.app/internal/sys"
	"hakurei.app/system"
)

func NewWithID(id state.ID, os sys.State) *App {
	a := new(App)
	a.id = newID(&id)
	a.sys = os
	return a
}

func AppIParams(a *App, sa SealedApp) (*system.I, *container.Params) {
	seal := sa.(*outcome)
	if a.outcome != seal || a.id != seal.id {
		panic("broken app/outcome link")
	}
	return seal.sys, seal.container
}
