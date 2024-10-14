package bwrap

const (
	Hostname = iota
	Chdir
	UnsetEnv
	LockFile

	RemountRO
	Procfs
	DevTmpfs
	Mqueue
)

var stringArgs = [...]string{
	Hostname: "--hostname",
	Chdir:    "--chdir",
	UnsetEnv: "--unsetenv",
	LockFile: "--lock-file",

	RemountRO: "--remount-ro",
	Procfs:    "--proc",
	DevTmpfs:  "--dev",
	Mqueue:    "--mqueue",
}

func (c *Config) stringArgs() Builder {
	n := stringArg{
		UnsetEnv: c.UnsetEnv,
		LockFile: c.LockFile,
	}

	if c.Hostname != "" {
		n[Hostname] = []string{c.Hostname}
	}
	if c.Chdir != "" {
		n[Chdir] = []string{c.Chdir}
	}

	// Arg types:
	// 	 RemountRO
	//   Procfs
	//   DevTmpfs
	//   Mqueue
	// are handled by the sequential builder

	return &n
}

type stringArg [len(stringArgs)][]string

func (s *stringArg) Len() (l int) {
	for _, arg := range s {
		l += len(arg) * 2
	}
	return
}

func (s *stringArg) Append(args *[]string) {
	for i, arg := range s {
		for _, v := range arg {
			*args = append(*args, stringArgs[i], v)
		}
	}
}
