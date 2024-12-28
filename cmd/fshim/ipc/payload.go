package shim0

import (
	"git.gensokyo.uk/security/fortify/helper/bwrap"
)

const Env = "FORTIFY_SHIM"

type Payload struct {
	// child full argv
	Argv []string
	// bwrap, target full exec path
	Exec [2]string
	// bwrap config
	Bwrap *bwrap.Config
	// path to outer home directory
	Home string
	// sync fd
	Sync *uintptr

	// verbosity pass through
	Verbose bool
}
