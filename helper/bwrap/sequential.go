package bwrap

import (
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"strconv"

	"git.gensokyo.uk/security/fortify/helper/proc"
)

func init() {
	gob.Register(new(PermConfig[SymlinkConfig]))
	gob.Register(new(PermConfig[*TmpfsConfig]))
	gob.Register(new(OverlayConfig))
	gob.Register(new(DataConfig))
}

type PositionalArg int

func (p PositionalArg) String() string { return positionalArgs[p] }

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

	SyncFd
	Seccomp

	File
	BindData
	ROBindData
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

	SyncFd:  "--sync-fd",
	Seccomp: "--seccomp",

	File:       "--file",
	BindData:   "--bind-data",
	ROBindData: "--ro-bind-data",
}

type PermConfig[T FSBuilder] struct {
	// set permissions of next argument
	// (--perms OCTAL)
	Mode *os.FileMode `json:"mode,omitempty"`
	// path to get the new permission
	// (--bind-data, --file, etc.)
	Inner T `json:"path"`
}

func (p *PermConfig[T]) Path() string { return p.Inner.Path() }

func (p *PermConfig[T]) Len() int {
	if p.Mode != nil {
		return p.Inner.Len() + 2
	} else {
		return p.Inner.Len()
	}
}

func (p *PermConfig[T]) Append(args *[]string) {
	if p.Mode != nil {
		*args = append(*args, Perms.String(), strconv.FormatInt(int64(*p.Mode), 8))
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

func (t *TmpfsConfig) Path() string { return t.Dir }

func (t *TmpfsConfig) Len() int {
	if t.Size > 0 {
		return 4
	} else {
		return 2
	}
}

func (t *TmpfsConfig) Append(args *[]string) {
	if t.Size > 0 {
		*args = append(*args, Size.String(), strconv.Itoa(t.Size))
	}
	*args = append(*args, Tmpfs.String(), t.Dir)
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

func (o *OverlayConfig) Path() string { return o.Dest }

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
		*args = append(*args, OverlaySrc.String(), src)
	}

	if o.Persist != nil {
		if o.Persist[0] != "" && o.Persist[1] != "" {
			// --overlay RWSRC WORKDIR
			*args = append(*args, Overlay.String(), o.Persist[0], o.Persist[1])
		} else {
			// --ro-overlay
			*args = append(*args, ROOverlay.String())
		}
	} else {
		// --tmp-overlay
		*args = append(*args, TmpOverlay.String())
	}

	// DEST
	*args = append(*args, o.Dest)
}

type SymlinkConfig [2]string

func (s SymlinkConfig) Path() string          { return s[1] }
func (s SymlinkConfig) Len() int              { return 3 }
func (s SymlinkConfig) Append(args *[]string) { *args = append(*args, Symlink.String(), s[0], s[1]) }

type ChmodConfig map[string]os.FileMode

func (c ChmodConfig) Len() int { return len(c) }
func (c ChmodConfig) Append(args *[]string) {
	for path, mode := range c {
		*args = append(*args, Chmod.String(), strconv.FormatInt(int64(mode), 8), path)
	}
}

const (
	DataWrite = iota
	DataBind
	DataROBind
)

type DataConfig struct {
	Dest string `json:"dest"`
	Data []byte `json:"data,omitempty"`
	Type int    `json:"type"`
	proc.File
}

func (d *DataConfig) Path() string { return d.Dest }
func (d *DataConfig) Len() int {
	if d == nil || d.Data == nil {
		return 0
	}
	return 3
}
func (d *DataConfig) Init(fd uintptr, v **os.File) uintptr {
	if d.File != nil {
		panic("file initialised twice")
	}
	d.File = proc.NewWriterTo(d)
	return d.File.Init(fd, v)
}
func (d *DataConfig) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(d.Data)
	return int64(n), err
}
func (d *DataConfig) Append(args *[]string) {
	if d == nil || d.Data == nil {
		return
	}
	var a PositionalArg
	switch d.Type {
	case DataWrite:
		a = File
	case DataBind:
		a = BindData
	case DataROBind:
		a = ROBindData
	default:
		panic(fmt.Sprintf("invalid type %d", a))
	}

	*args = append(*args, a.String(), strconv.Itoa(int(d.Fd())), d.Dest)
}
