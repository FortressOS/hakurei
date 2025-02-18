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
	a.os = os
	return a
}

func AppSystemBwrap(a fst.App) (*system.I, *bwrap.Config) {
	v := a.(*app)
	return v.seal.sys.I, v.seal.sys.bwrap
}
