package bwrap

const (
	SetEnv = iota

	Bind
	BindTry
	DevBind
	DevBindTry
	ROBind
	ROBindTry

	Chmod
)

var pairArgs = [...]string{
	SetEnv: "--setenv",

	Bind:       "--bind",
	BindTry:    "--bind-try",
	DevBind:    "--dev-bind",
	DevBindTry: "--dev-bind-try",
	ROBind:     "--ro-bind",
	ROBindTry:  "--ro-bind-try",

	Chmod: "--chmod",
}

func (c *Config) pairArgs() Builder {
	var n pairArg
	n[SetEnv] = make([][2]string, 0, len(c.SetEnv))
	for k, v := range c.SetEnv {
		n[SetEnv] = append(n[SetEnv], [2]string{k, v})
	}

	// Arg types:
	//   Bind
	//   BindTry
	//   DevBind
	//   DevBindTry
	//   ROBind
	//   ROBindTry
	//   Chmod
	// are handled by the sequential builder

	return &n
}

type pairArg [len(pairArgs)][][2]string

func (p *pairArg) Len() (l int) {
	for _, v := range p {
		l += len(v) * 3
	}
	return
}

func (p *pairArg) Append(args *[]string) {
	for i, arg := range p {
		for _, v := range arg {
			*args = append(*args, pairArgs[i], v[0], v[1])
		}
	}
}
