package shim0

import (
	"git.ophivana.moe/security/fortify/helper/bwrap"
)

const Env = "FORTIFY_SHIM"

type Payload struct {
	// child full argv
	Argv []string
	// bwrap, target full exec path
	Exec [2]string
	// bwrap config
	Bwrap *bwrap.Config
	// sync fd
	Sync *uintptr

	// verbosity pass through
	Verbose bool
}
