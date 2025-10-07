package hst

import (
	"encoding/gob"
	"os"
	"strings"

	"hakurei.app/container/check"
)

func init() { gob.Register(new(FSEphemeral)) }

// FilesystemEphemeral is the type string of a mount point with ephemeral state.
const FilesystemEphemeral = "ephemeral"

// FSEphemeral represents an ephemeral container mount point.
type FSEphemeral struct {
	// mount point in container
	Target *check.Absolute `json:"dst,omitempty"`
	// do not mount filesystem read-only
	Write bool `json:"write,omitempty"`
	// upper limit on the size of the filesystem
	Size int `json:"size,omitempty"`
	// initial permission bits of the new filesystem
	Perm os.FileMode `json:"perm,omitempty"`
}

func (e *FSEphemeral) Valid() bool { return e != nil && e.Target != nil }

func (e *FSEphemeral) Path() *check.Absolute {
	if !e.Valid() {
		return nil
	}
	return e.Target
}

func (e *FSEphemeral) Host() []*check.Absolute { return nil }

const fsEphemeralDefaultPerm = os.FileMode(0755)

func (e *FSEphemeral) Apply(z *ApplyState) {
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
		z.Tmpfs(e.Target, size, perm)
	} else {
		z.Readonly(e.Target, perm)
	}
}

func (e *FSEphemeral) String() string {
	if !e.Valid() {
		return "<invalid>"
	}

	expr := new(strings.Builder)
	expr.Grow(15 + len(FilesystemEphemeral) + len(e.Target.String()))

	if e.Write {
		expr.WriteString("w")
	}
	expr.WriteString("+" + FilesystemEphemeral + "(")
	if e.Perm != 0 {
		expr.WriteString(e.Perm.String())
	} else {
		expr.WriteString(fsEphemeralDefaultPerm.String())
	}
	expr.WriteString("):" + e.Target.String())

	return expr.String()
}
