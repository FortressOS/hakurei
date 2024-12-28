package app

import (
	"git.gensokyo.uk/security/fortify/helper/bwrap"
	"git.gensokyo.uk/security/fortify/internal/system"
)

// appSealSys encapsulates app seal behaviour with OS interactions
type appSealSys struct {
	bwrap *bwrap.Config
	// paths to override by mounting tmpfs over them
	override []string

	// default formatted XDG_RUNTIME_DIR of User
	runtime string
	// target user sealed from config
	user appUser

	// mapped uid and gid in user namespace
	mappedID int
	// string representation of mappedID
	mappedIDString string

	needRevert bool
	saveState  bool
	*system.I

	// protected by upstream mutex
}

type appUser struct {
	// full uid resolved by fsu
	uid int
	// string representation of uid
	us string

	// supplementary group ids
	supp []string

	// application id
	aid int
	// string representation of aid
	as string

	// home directory host path
	data string
	// app user home directory
	home string
	// passwd database username
	username string
}
