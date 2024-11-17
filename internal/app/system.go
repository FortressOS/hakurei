package app

import (
	"git.ophivana.moe/security/fortify/dbus"
	"git.ophivana.moe/security/fortify/helper/bwrap"
	"git.ophivana.moe/security/fortify/internal/linux"
	"git.ophivana.moe/security/fortify/internal/system"
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

// shareAll calls all share methods in sequence
func (seal *appSeal) shareAll(bus [2]*dbus.Config, os linux.System) error {
	if seal.shared {
		panic("seal shared twice")
	}
	seal.shared = true

	seal.shareSystem()
	seal.shareRuntime()
	seal.sharePasswd(os)
	if err := seal.shareDisplay(os); err != nil {
		return err
	}
	if err := seal.sharePulse(os); err != nil {
		return err
	}

	// ensure dbus session bus defaults
	if bus[0] == nil {
		bus[0] = dbus.NewConfig(seal.fid, true, true)
	}

	if err := seal.shareDBus(bus); err != nil {
		return err
	}

	// queue overriding tmpfs at the end of seal.sys.bwrap.Filesystem
	for _, dest := range seal.sys.override {
		seal.sys.bwrap.Tmpfs(dest, 8*1024)
	}

	return nil
}
