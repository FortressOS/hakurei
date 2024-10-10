package shim

import (
	"git.ophivana.moe/cat/fortify/helper/bwrap"
)

const EnvShim = "FORTIFY_SHIM"

type Payload struct {
	// child full argv
	Argv []string
	// env variables passed through to bwrap
	Env []string
	// bwrap, target full exec path
	Exec [2]string
	// bwrap config, nil for permissive
	Bwrap *bwrap.Config
	// whether to pas wayland fd
	WL bool

	// verbosity pass through
	Verbose bool
}
