package hst

import (
	"encoding/gob"
	"strings"

	"hakurei.app/container"
)

func init() { gob.Register(new(FSBind)) }

// FilesystemBind is the [FilesystemConfig.Type] name of a bind mount point.
const FilesystemBind = "bind"

// FSBind represents a host to container bind mount.
type FSBind struct {
	// mount point in container, same as src if empty
	Target *container.Absolute `json:"dst,omitempty"`
	// host filesystem path to make available to the container
	Source *container.Absolute `json:"src"`
	// do not mount filesystem read-only
	Write bool `json:"write,omitempty"`
	// do not disable device files, implies Write
	Device bool `json:"dev,omitempty"`
	// skip this mount point if the host path does not exist
	Optional bool `json:"optional,omitempty"`
}

func (b *FSBind) Valid() bool { return b != nil && b.Source != nil }

func (b *FSBind) Path() *container.Absolute {
	if !b.Valid() {
		return nil
	}
	if b.Target == nil {
		return b.Source
	}
	return b.Target
}

func (b *FSBind) Host() []*container.Absolute {
	if !b.Valid() {
		return nil
	}
	return []*container.Absolute{b.Source}
}

func (b *FSBind) Apply(ops *container.Ops) {
	if !b.Valid() {
		return
	}

	target := b.Target
	if target == nil {
		target = b.Source
	}
	var flags int
	if b.Write {
		flags |= container.BindWritable
	}
	if b.Device {
		flags |= container.BindDevice | container.BindWritable
	}
	if b.Optional {
		flags |= container.BindOptional
	}
	ops.Bind(b.Source, target, flags)
}

func (b *FSBind) String() string {
	g := 4
	if !b.Valid() {
		return "<invalid>"
	}

	g += len(b.Source.String())
	if b.Target != nil {
		g += len(b.Target.String())
	}

	expr := new(strings.Builder)
	expr.Grow(g)

	if b.Device {
		expr.WriteString("d")
	} else if b.Write {
		expr.WriteString("w")
	}

	if !b.Optional {
		expr.WriteString("*")
	} else {
		expr.WriteString("+")
	}

	expr.WriteString(b.Source.String())
	if b.Target != nil {
		expr.WriteString(":" + b.Target.String())
	}

	return expr.String()
}
