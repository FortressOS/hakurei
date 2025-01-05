package bwrap

const (
	Tmpfs = iota
	Dir
	Symlink

	OverlaySrc
	Overlay
	TmpOverlay
	ROOverlay
)

var awkwardArgs = [...]string{
	Tmpfs:   "--tmpfs",
	Dir:     "--dir",
	Symlink: "--symlink",

	OverlaySrc: "--overlay-src",
	Overlay:    "--overlay",
	TmpOverlay: "--tmp-overlay",
	ROOverlay:  "--ro-overlay",
}
