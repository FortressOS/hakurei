package app

import (
	"os"

	"git.gensokyo.uk/security/fortify/helper/bwrap"
	"git.gensokyo.uk/security/fortify/system"
)

// appSealSys encapsulates app seal behaviour with OS interactions
type appSealSys struct {
	bwrap *bwrap.Config
	// bwrap sync fd
	sp *os.File
	// paths to override by mounting tmpfs over them
	override []string

	// default formatted XDG_RUNTIME_DIR of User
	runtime string
	// target user sealed from config
	user appUser

	// mapped uid and gid in user namespace
	mapuid *stringPair[int]

	needRevert bool
	saveState  bool
	*system.I

	// protected by upstream mutex
}

type appUser struct {
	// application id
	aid *stringPair[int]
	// target uid resolved by fid:aid
	uid *stringPair[int]

	// supplementary group ids
	supp []string

	// home directory host path
	data string
	// app user home directory
	home string
	// passwd database username
	username string
}
