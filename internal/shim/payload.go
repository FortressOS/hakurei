package shim

import "git.ophivana.moe/cat/fortify/helper/bwrap"

const EnvShim = "FORTIFY_SHIM"

type Payload struct {
	// child full argv
	Argv []string
	// fortify, bwrap, target full exec path
	Exec [3]string
	// bwrap config
	Bwrap *bwrap.Config
	// whether to pass wayland fd
	WL bool

	// verbosity pass through
	Verbose bool
}
