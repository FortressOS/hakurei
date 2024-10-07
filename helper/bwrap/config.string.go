package bwrap

const (
	Hostname = iota
	Chdir
	UnsetEnv
	LockFile
	RemountRO

	stringC
)

var stringArgs = func() (n [stringC]string) {
	n[Hostname] = "--hostname"
	n[Chdir] = "--chdir"
	n[UnsetEnv] = "--unsetenv"
	n[LockFile] = "--lock-file"
	n[RemountRO] = "--remount-ro"

	return
}()

func (c *Config) stringArgs() (n [stringC][]string) {
	if c.Hostname != "" {
		n[Hostname] = []string{c.Hostname}
	}
	if c.Chdir != "" {
		n[Chdir] = []string{c.Chdir}
	}
	n[UnsetEnv] = c.UnsetEnv
	n[LockFile] = c.LockFile
	n[RemountRO] = c.RemountRO

	return
}
