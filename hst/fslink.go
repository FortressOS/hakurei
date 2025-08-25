package hst

import (
	"encoding/gob"
	"path"

	"hakurei.app/container"
)

func init() { gob.Register(new(FSLink)) }

// FilesystemLink is the type string of a symbolic link.
const FilesystemLink = "link"

// FSLink represents a symlink in the container filesystem.
type FSLink struct {
	// link path in container
	Target *container.Absolute `json:"dst"`
	// linkname the symlink points to
	Linkname string `json:"linkname"`
	// whether to dereference linkname before creating the link
	Dereference bool `json:"dereference,omitempty"`
}

func (l *FSLink) Valid() bool {
	if l == nil || l.Target == nil || l.Linkname == "" {
		return false
	}
	return !l.Dereference || path.IsAbs(l.Linkname)
}

func (l *FSLink) Path() *container.Absolute {
	if !l.Valid() {
		return nil
	}
	return l.Target
}

func (l *FSLink) Host() []*container.Absolute { return nil }

func (l *FSLink) Apply(z *ApplyState) {
	if !l.Valid() {
		return
	}
	z.Link(l.Target, l.Linkname, l.Dereference)
}

func (l *FSLink) String() string {
	if !l.Valid() {
		return "<invalid>"
	}

	var dereference string
	if l.Dereference {
		if l.Target.String() == l.Linkname {
			return l.Target.String() + "@"
		}
		dereference = "*"
	}
	return l.Target.String() + " -> " + dereference + l.Linkname
}
