package shim

import (
	"git.ophivana.moe/cat/fortify/helper/bwrap"
)

const EnvShim = "FORTIFY_SHIM"

type Payload struct {
	// child full argv
	Argv []string
	// bwrap, target full exec path
	Exec [2]string
	// bwrap config, nil for permissive
	Bwrap *bwrap.Config
	// whether to pass wayland fd
	WL bool

	// verbosity pass through
	Verbose bool
}
