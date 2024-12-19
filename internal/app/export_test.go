package app

import (
	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/helper/bwrap"
	"git.gensokyo.uk/security/fortify/internal/linux"
	"git.gensokyo.uk/security/fortify/internal/system"
)

func NewWithID(id fst.ID, os linux.System) App {
	a := new(app)
	a.id = &id
	a.os = os
	return a
}

func AppSystemBwrap(a App) (*system.I, *bwrap.Config) {
	v := a.(*app)
	return v.seal.sys.I, v.seal.sys.bwrap
}
