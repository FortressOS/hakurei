package hst

import (
	"encoding/gob"
	"strings"

	"hakurei.app/container"
	"hakurei.app/container/check"
	"hakurei.app/container/fhs"
)

func init() { gob.Register(new(FSBind)) }

// FilesystemBind is the type string of a bind mount point.
const FilesystemBind = "bind"

// FSBind represents a host to container bind mount.
type FSBind struct {
	// mount point in container, same as Source if empty
	Target *check.Absolute `json:"dst,omitempty"`
	// host filesystem path to make available to the container
	Source *check.Absolute `json:"src"`
	// do not mount Target read-only
	Write bool `json:"write,omitempty"`
	// do not disable device files on Target, implies Write
	Device bool `json:"dev,omitempty"`
	// create Source as a directory if it does not exist
	Ensure bool `json:"ensure,omitempty"`
	// skip this mount point if Source does not exist
	Optional bool `json:"optional,omitempty"`

	// enable special behaviour:
	// for autoroot, Target must be set to [fhs.AbsRoot];
	// for autoetc, Target must be set to [fhs.AbsEtc]
	Special bool `json:"special,omitempty"`
}

// IsAutoRoot returns whether this FSBind has autoroot behaviour enabled.
func (b *FSBind) IsAutoRoot() bool {
	return b.Valid() && b.Special && b.Target.String() == fhs.Root
}

// IsAutoEtc returns whether this FSBind has autoetc behaviour enabled.
func (b *FSBind) IsAutoEtc() bool {
	return b.Valid() && b.Special && b.Target.String() == fhs.Etc
}

func (b *FSBind) Valid() bool {
	if b == nil || b.Source == nil {
		return false
	}
	if b.Ensure && b.Optional {
		return false
	}
	if b.Special {
		if b.Target == nil {
			return false
		} else {
			switch b.Target.String() {
			case fhs.Root, fhs.Etc:
				break

			default:
				return false
			}
		}
	}
	return true
}

func (b *FSBind) Path() *check.Absolute {
	if !b.Valid() {
		return nil
	}
	if b.Target == nil {
		return b.Source
	}
	return b.Target
}

func (b *FSBind) Host() []*check.Absolute {
	if !b.Valid() {
		return nil
	}
	return []*check.Absolute{b.Source}
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
	if b.Ensure {
		flags |= container.BindEnsure
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
			if b.Source.String() != fhs.Root {
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

	switch {
	case b.Ensure:
		expr.WriteString("-")

	case b.Optional:
		expr.WriteString("+")

	default:
		expr.WriteString("*")
	}

	expr.WriteString(b.Source.String())
	if b.Target != nil {
		expr.WriteString(":" + b.Target.String())
	}

	return expr.String()
}
