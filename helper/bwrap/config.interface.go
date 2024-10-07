package bwrap

const (
	Procfs = iota
	DevTmpfs
	Tmpfs
	Mqueue
	Dir
	Symlink

	interfaceC
)

var interfaceArgs = func() (g [interfaceC]string) {
	g[Procfs] = "--proc"
	g[DevTmpfs] = "--dev"
	g[Tmpfs] = "--tmpfs"
	g[Mqueue] = "--mqueue"
	g[Dir] = "--dir"
	g[Symlink] = "--symlink"

	return
}()

func (c *Config) interfaceArgs() (g [interfaceC][]argOf) {
	g[Procfs] = copyToArgOfSlice(c.Procfs)
	g[DevTmpfs] = copyToArgOfSlice(c.DevTmpfs)
	g[Tmpfs] = copyToArgOfSlice(c.Tmpfs)
	g[Mqueue] = copyToArgOfSlice(c.Mqueue)
	g[Dir] = copyToArgOfSlice(c.Dir)
	g[Symlink] = copyToArgOfSlice(c.Symlink)

	return
}
