package container

import (
	"encoding/gob"
	"fmt"
	"os"
	"syscall"
)

func init() { gob.Register(new(AutoRootOp)) }

// Root appends an [Op] that expands a directory into a toplevel bind mount mirror on container root.
// This is not a generic setup op. It is implemented here to reduce ipc overhead.
func (f *Ops) Root(host *Absolute, prefix string, flags int) *Ops {
	*f = append(*f, &AutoRootOp{host, prefix, flags, nil})
	return f
}

type AutoRootOp struct {
	Host   *Absolute
	Prefix string
	// passed through to bindMount
	Flags int

	// obtained during early;
	// these wrap the underlying Op because BindMountOp is relatively complex,
	// so duplicating that code would be unwise
	resolved []Op
}

func (r *AutoRootOp) early(params *Params) error {
	if r.Host == nil {
		return syscall.EBADE
	}

	if d, err := os.ReadDir(r.Host.String()); err != nil {
		return wrapErrSelf(err)
	} else {
		r.resolved = make([]Op, 0, len(d))
		for _, ent := range d {
			name := ent.Name()
			if IsAutoRootBindable(name) {
				op := &BindMountOp{
					Source: r.Host.Append(name),
					Target: AbsFHSRoot.Append(name),
					Flags:  r.Flags,
				}
				if err = op.early(params); err != nil {
					return err
				}
				r.resolved = append(r.resolved, op)
			}
		}
		return nil
	}
}

func (r *AutoRootOp) apply(params *Params) error {
	for _, op := range r.resolved {
		msg.Verbosef("%s %s", op.prefix(), op)
		if err := op.apply(params); err != nil {
			return err
		}
	}
	return nil
}

func (r *AutoRootOp) Is(op Op) bool {
	vr, ok := op.(*AutoRootOp)
	return ok && ((r == nil && vr == nil) || (r != nil && vr != nil &&
		r.Host == vr.Host && r.Prefix == vr.Prefix && r.Flags == vr.Flags))
}
func (*AutoRootOp) prefix() string { return "setting up" }
func (r *AutoRootOp) String() string {
	return fmt.Sprintf("auto root %q prefix %s flags %#x", r.Host, r.Prefix, r.Flags)
}

// IsAutoRootBindable returns whether a dir entry name is selected for AutoRoot.
func IsAutoRootBindable(name string) bool {
	switch name {
	case "proc":
	case "dev":
	case "tmp":
	case "mnt":
	case "etc":

	default:
		return true
	}
	return false
}
