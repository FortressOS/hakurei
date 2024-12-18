package app

import (
	"git.ophivana.moe/security/fortify/fst"
	"git.ophivana.moe/security/fortify/helper/bwrap"
	"git.ophivana.moe/security/fortify/internal/linux"
	"git.ophivana.moe/security/fortify/internal/system"
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
