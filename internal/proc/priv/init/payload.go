package init0

const Env = "FORTIFY_INIT"

type Payload struct {
	// target full exec path
	Argv0 string
	// child full argv
	Argv []string

	// verbosity pass through
	Verbose bool
}
