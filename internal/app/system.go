package app

import (
	"os/user"

	"git.ophivana.moe/security/fortify/dbus"
	"git.ophivana.moe/security/fortify/helper/bwrap"
	"git.ophivana.moe/security/fortify/internal"
	"git.ophivana.moe/security/fortify/internal/state"
	"git.ophivana.moe/security/fortify/internal/system"
)

// appSeal seals the application with child-related information
type appSeal struct {
	// wayland socket path if mediated wayland is enabled
	wl string
	// wait for wayland client to exit if mediated wayland is enabled,
	// (wlDone == nil) determines whether mediated wayland setup is performed
	wlDone chan struct{}

	// app unique ID string representation
	id string
	// freedesktop application ID
	fid string
	// argv to start process with in the final confined environment
	command []string
	// persistent process state store
	store state.Store

	// uint8 representation of launch method sealed from config
	launchOption uint8
	// process-specific share directory path
	share string
	// process-specific share directory path local to XDG_RUNTIME_DIR
	shareLocal string

	// path to launcher program
	toolPath string
	// pass-through enablement tracking from config
	et system.Enablements

	// prevents sharing from happening twice
	shared bool
	// seal system-level component
	sys *appSealSys

	// used in various sealing operations
	internal.SystemConstants

	// protected by upstream mutex
}

// appSealSys encapsulates app seal behaviour with OS interactions
type appSealSys struct {
	bwrap *bwrap.Config
	// paths to override by mounting tmpfs over them
	override []string

	// default formatted XDG_RUNTIME_DIR of User
	runtime string
	// sealed path to fortify executable, used by shim
	executable string
	// target user sealed from config
	user *user.User

	*system.I

	// protected by upstream mutex
}

// shareAll calls all share methods in sequence
func (seal *appSeal) shareAll(bus [2]*dbus.Config) error {
	if seal.shared {
		panic("seal shared twice")
	}
	seal.shared = true

	seal.shareSystem()
	seal.shareRuntime()
	seal.sharePasswd()
	if err := seal.shareDisplay(); err != nil {
		return err
	}
	if err := seal.sharePulse(); err != nil {
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
