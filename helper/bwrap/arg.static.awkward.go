package bwrap

const (
	Tmpfs = iota
	Dir
	Symlink
)

var awkwardArgs = [...]string{
	Tmpfs:   "--tmpfs",
	Dir:     "--dir",
	Symlink: "--symlink",
}
