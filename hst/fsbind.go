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

	// enable special behaviour:
	// for autoroot, Target must be set to [container.AbsFHSRoot];
	// for autoetc, Target must be set to [container.AbsFHSEtc]
	Special bool `json:"special,omitempty"`
}

// IsAutoRoot returns whether this FSBind has autoroot behaviour enabled.
func (b *FSBind) IsAutoRoot() bool {
	return b.Valid() && b.Special && b.Target.String() == container.FHSRoot
}

// IsAutoEtc returns whether this FSBind has autoetc behaviour enabled.
func (b *FSBind) IsAutoEtc() bool {
	return b.Valid() && b.Special && b.Target.String() == container.FHSEtc
}

func (b *FSBind) Valid() bool {
	if b == nil || b.Source == nil {
		return false
	}
	if b.Special {
		if b.Target == nil {
			return false
		} else {
			switch b.Target.String() {
			case container.FHSRoot, container.FHSEtc:
				break

			default:
				return false
			}
		}
	}
	return true
}

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

func (b *FSBind) Apply(z *ApplyState) {
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

	switch {
	case b.IsAutoRoot():
		z.Root(b.Source, flags)

	case b.IsAutoEtc():
		z.Etc(b.Source, z.AutoEtcPrefix)

	default:
		z.Bind(b.Source, target, flags)
	}
}

func (b *FSBind) String() string {
	if !b.Valid() {
		return "<invalid>"
	}

	var flagSym string
	if b.Device {
		flagSym = "d"
	} else if b.Write {
		flagSym = "w"
	}

	if b.Special {
		switch {
		case b.IsAutoRoot():
			prefix := "autoroot"
			if flagSym != "" {
				prefix += ":" + flagSym
			}
			if b.Source.String() != container.FHSRoot {
				return prefix + ":" + b.Source.String()
			}
			return prefix

		case b.IsAutoEtc():
			return "autoetc:" + b.Source.String()
		}
	}

	g := 4 + len(b.Source.String())
	if b.Target != nil {
		g += len(b.Target.String())
	}

	expr := new(strings.Builder)
	expr.Grow(g)
	expr.WriteString(flagSym)

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
