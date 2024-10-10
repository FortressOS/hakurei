package bwrap

const (
	Tmpfs = iota
	Dir
	Symlink

	interfaceC
)

var interfaceArgs = func() (g [interfaceC]string) {
	g[Tmpfs] = "--tmpfs"
	g[Dir] = "--dir"
	g[Symlink] = "--symlink"

	return
}()

func (c *Config) interfaceArgs() (g [interfaceC][]argOf) {
	g[Tmpfs] = copyToArgOfSlice(c.Tmpfs)
	g[Dir] = copyToArgOfSlice(c.Dir)
	g[Symlink] = copyToArgOfSlice(c.Symlink)

	return
}
