package app

import (
	"hakurei.app/container"
	"hakurei.app/internal/app/state"
	"hakurei.app/internal/sys"
	"hakurei.app/system"
)

func NewWithID(id state.ID, os sys.State) App {
	a := new(app)
	a.id = newID(&id)
	a.sys = os
	return a
}

func AppIParams(a App, sa SealedApp) (*system.I, *container.Params) {
	v := a.(*app)
	seal := sa.(*outcome)
	if v.outcome != seal || v.id != seal.id {
		panic("broken app/outcome link")
	}
	return seal.sys, seal.container
}
