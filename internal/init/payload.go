package init0

const EnvInit = "FORTIFY_INIT"

type Payload struct {
	// target full exec path
	Argv0 string
	// child full argv
	Argv []string
	// wayland fd, -1 to disable
	WL int

	// verbosity pass through
	Verbose bool
}
