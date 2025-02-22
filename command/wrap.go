package command

// the top level node wants [Command] returned for its builder methods
type rootNode struct{ *node }

func (r rootNode) Command(name, usage string, f HandlerFunc) Command {
	r.node.Command(name, usage, f)
	return r
}

func (r rootNode) Flag(p any, name string, value FlagDefiner, usage string) Command {
	r.node.Flag(p, name, value, usage)
	return r
}
