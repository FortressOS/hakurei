package app

import (
	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/helper/bwrap"
	"git.gensokyo.uk/security/fortify/internal/sys"
	"git.gensokyo.uk/security/fortify/system"
)

func NewWithID(id fst.ID, os sys.State) fst.App {
	a := new(app)
	a.id = newID(&id)
	a.sys = os
	return a
}

func AppSystemBwrap(a fst.App, sa fst.SealedApp) (*system.I, *bwrap.Config) {
	v := a.(*app)
	seal := sa.(*outcome)
	if v.outcome != seal || v.id != seal.id {
		panic("broken app/outcome link")
	}
	return seal.sys, seal.container
}
