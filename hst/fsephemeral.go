package hst

import (
	"encoding/gob"
	"os"
	"strings"

	"hakurei.app/container"
)

func init() { gob.Register(new(FSEphemeral)) }

// FilesystemEphemeral is the [FilesystemConfig.Type] name of a mount point with ephemeral state.
const FilesystemEphemeral = "ephemeral"

// FSEphemeral represents an ephemeral container mount point.
type FSEphemeral struct {
	// mount point in container
	Dst *container.Absolute `json:"dst,omitempty"`
	// do not mount filesystem read-only
	Write bool `json:"write,omitempty"`
	// upper limit on the size of the filesystem
	Size int `json:"size,omitempty"`
	// initial permission bits of the new filesystem
	Perm os.FileMode `json:"perm,omitempty"`
}

func (e *FSEphemeral) Valid() bool { return e != nil && e.Dst != nil }

func (e *FSEphemeral) Target() *container.Absolute {
	if !e.Valid() {
		return nil
	}
	return e.Dst
}

func (e *FSEphemeral) Host() []*container.Absolute { return nil }

const fsEphemeralDefaultPerm = os.FileMode(0755)

func (e *FSEphemeral) Apply(ops *container.Ops) {
	if !e.Valid() {
		return
	}

	size := e.Size
	if size < 0 {
		size = 0
	}

	perm := e.Perm
	if perm == 0 {
		perm = fsEphemeralDefaultPerm
	}

	if e.Write {
		ops.Tmpfs(e.Dst, size, perm)
	} else {
		ops.Readonly(e.Dst, perm)
	}
}

func (e *FSEphemeral) String() string {
	if !e.Valid() {
		return "<invalid>"
	}

	expr := new(strings.Builder)
	expr.Grow(15 + len(FilesystemEphemeral) + len(e.Dst.String()))

	if e.Write {
		expr.WriteString("w")
	}
	expr.WriteString("+" + FilesystemEphemeral + "(")
	if e.Perm != 0 {
		expr.WriteString(e.Perm.String())
	} else {
		expr.WriteString(fsEphemeralDefaultPerm.String())
	}
	expr.WriteString("):" + e.Dst.String())

	return expr.String()
}
