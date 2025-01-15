package bwrap

import (
	"encoding/gob"
	"os"
	"strconv"
)

func init() {
	gob.Register(new(PermConfig[SymlinkConfig]))
	gob.Register(new(PermConfig[*TmpfsConfig]))
	gob.Register(new(OverlayConfig))
}

type PositionalArg int

func (p PositionalArg) Unwrap() string {
	return positionalArgs[p]
}

const (
	Tmpfs PositionalArg = iota
	Symlink

	Bind
	BindTry
	DevBind
	DevBindTry
	ROBind
	ROBindTry

	Chmod
	Dir
	RemountRO
	Procfs
	DevTmpfs
	Mqueue

	Perms
	Size

	OverlaySrc
	Overlay
	TmpOverlay
	ROOverlay
)

var positionalArgs = [...]string{
	Tmpfs:   "--tmpfs",
	Symlink: "--symlink",

	Bind:       "--bind",
	BindTry:    "--bind-try",
	DevBind:    "--dev-bind",
	DevBindTry: "--dev-bind-try",
	ROBind:     "--ro-bind",
	ROBindTry:  "--ro-bind-try",

	Chmod:     "--chmod",
	Dir:       "--dir",
	RemountRO: "--remount-ro",
	Procfs:    "--proc",
	DevTmpfs:  "--dev",
	Mqueue:    "--mqueue",

	Perms: "--perms",
	Size:  "--size",

	OverlaySrc: "--overlay-src",
	Overlay:    "--overlay",
	TmpOverlay: "--tmp-overlay",
	ROOverlay:  "--ro-overlay",
}

type PermConfig[T FSBuilder] struct {
	// set permissions of next argument
	// (--perms OCTAL)
	Mode *os.FileMode `json:"mode,omitempty"`
	// path to get the new permission
	// (--bind-data, --file, etc.)
	Inner T `json:"path"`
}

func (p *PermConfig[T]) Path() string {
	return p.Inner.Path()
}

func (p *PermConfig[T]) Len() int {
	if p.Mode != nil {
		return p.Inner.Len() + 2
	} else {
		return p.Inner.Len()
	}
}

func (p *PermConfig[T]) Append(args *[]string) {
	if p.Mode != nil {
		*args = append(*args, Perms.Unwrap(), strconv.FormatInt(int64(*p.Mode), 8))
	}
	p.Inner.Append(args)
}

type TmpfsConfig struct {
	// set size of tmpfs
	// (--size BYTES)
	Size int `json:"size,omitempty"`
	// mount point of new tmpfs
	// (--tmpfs DEST)
	Dir string `json:"dir"`
}

func (t *TmpfsConfig) Path() string {
	return t.Dir
}

func (t *TmpfsConfig) Len() int {
	if t.Size > 0 {
		return 4
	} else {
		return 2
	}
}

func (t *TmpfsConfig) Append(args *[]string) {
	if t.Size > 0 {
		*args = append(*args, Size.Unwrap(), strconv.Itoa(t.Size))
	}
	*args = append(*args, Tmpfs.Unwrap(), t.Dir)
}

type OverlayConfig struct {
	/*
		read files from SRC in the following overlay
		(--overlay-src SRC)
	*/
	Src []string `json:"src,omitempty"`

	/*
		mount overlayfs on DEST, with RWSRC as the host path for writes and
		WORKDIR an empty directory on the same filesystem as RWSRC
		(--overlay RWSRC WORKDIR DEST)

		if nil, mount overlayfs on DEST, with writes going to an invisible tmpfs
		(--tmp-overlay DEST)

		if either strings are empty, mount overlayfs read-only on DEST
		(--ro-overlay DEST)
	*/
	Persist *[2]string `json:"persist,omitempty"`

	/*
		--overlay RWSRC WORKDIR DEST

		--tmp-overlay DEST

		--ro-overlay DEST
	*/
	Dest string `json:"dest"`
}

func (o *OverlayConfig) Path() string {
	return o.Dest
}

func (o *OverlayConfig) Len() int {
	// (--tmp-overlay DEST) or (--ro-overlay DEST)
	p := 2
	// (--overlay RWSRC WORKDIR DEST)
	if o.Persist != nil && o.Persist[0] != "" && o.Persist[1] != "" {
		p = 4
	}

	return p + len(o.Src)*2
}

func (o *OverlayConfig) Append(args *[]string) {
	// --overlay-src SRC
	for _, src := range o.Src {
		*args = append(*args, OverlaySrc.Unwrap(), src)
	}

	if o.Persist != nil {
		if o.Persist[0] != "" && o.Persist[1] != "" {
			// --overlay RWSRC WORKDIR
			*args = append(*args, Overlay.Unwrap(), o.Persist[0], o.Persist[1])
		} else {
			// --ro-overlay
			*args = append(*args, ROOverlay.Unwrap())
		}
	} else {
		// --tmp-overlay
		*args = append(*args, TmpOverlay.Unwrap())
	}

	// DEST
	*args = append(*args, o.Dest)
}

type SymlinkConfig [2]string

func (s SymlinkConfig) Path() string {
	return s[1]
}

func (s SymlinkConfig) Len() int {
	return 3
}

func (s SymlinkConfig) Append(args *[]string) {
	*args = append(*args, Symlink.Unwrap(), s[0], s[1])
}

type ChmodConfig map[string]os.FileMode

func (c ChmodConfig) Len() int {
	return len(c)
}

func (c ChmodConfig) Append(args *[]string) {
	for path, mode := range c {
		*args = append(*args, Chmod.Unwrap(), strconv.FormatInt(int64(mode), 8), path)
	}
}
