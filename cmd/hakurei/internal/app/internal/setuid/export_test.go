package setuid

import (
	"git.gensokyo.uk/security/hakurei"
	. "git.gensokyo.uk/security/hakurei/cmd/hakurei/internal/app"
	"git.gensokyo.uk/security/hakurei/internal/sys"
	"git.gensokyo.uk/security/hakurei/system"
)

func NewWithID(id ID, os sys.State) App {
	a := new(app)
	a.id = newID(&id)
	a.sys = os
	return a
}

func AppIParams(a App, sa SealedApp) (*system.I, *hakurei.Params) {
	v := a.(*app)
	seal := sa.(*outcome)
	if v.outcome != seal || v.id != seal.id {
		panic("broken app/outcome link")
	}
	return seal.sys, seal.container
}
