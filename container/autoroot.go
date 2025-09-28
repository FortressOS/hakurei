package container

import (
	"encoding/gob"
	"fmt"
)

func init() { gob.Register(new(AutoRootOp)) }

// Root appends an [Op] that expands a directory into a toplevel bind mount mirror on container root.
// This is not a generic setup op. It is implemented here to reduce ipc overhead.
func (f *Ops) Root(host *Absolute, flags int) *Ops {
	*f = append(*f, &AutoRootOp{host, flags, nil})
	return f
}

type AutoRootOp struct {
	Host *Absolute
	// passed through to bindMount
	Flags int

	// obtained during early;
	// these wrap the underlying Op because BindMountOp is relatively complex,
	// so duplicating that code would be unwise
	resolved []*BindMountOp
}

func (r *AutoRootOp) Valid() bool { return r != nil && r.Host != nil }

func (r *AutoRootOp) early(state *setupState, k syscallDispatcher) error {
	if d, err := k.readdir(r.Host.String()); err != nil {
		return err
	} else {
		r.resolved = make([]*BindMountOp, 0, len(d))
		for _, ent := range d {
			name := ent.Name()
			if IsAutoRootBindable(state, name) {
				// careful: the Valid method is skipped, make sure this is always valid
				op := &BindMountOp{
					Source: r.Host.Append(name),
					Target: AbsFHSRoot.Append(name),
					Flags:  r.Flags,
				}
				if err = op.early(state, k); err != nil {
					return err
				}
				r.resolved = append(r.resolved, op)
			}
		}
		return nil
	}
}

func (r *AutoRootOp) apply(state *setupState, k syscallDispatcher) error {
	if state.nonrepeatable&nrAutoRoot != 0 {
		return OpRepeatError("autoroot")
	}
	state.nonrepeatable |= nrAutoRoot

	for _, op := range r.resolved {
		// these are exclusively BindMountOp, do not attempt to print identifying message
		if err := op.apply(state, k); err != nil {
			return err
		}
	}
	return nil
}

func (r *AutoRootOp) Is(op Op) bool {
	vr, ok := op.(*AutoRootOp)
	return ok && r.Valid() && vr.Valid() &&
		r.Host.Is(vr.Host) &&
		r.Flags == vr.Flags
}
func (*AutoRootOp) prefix() (string, bool) { return "setting up", true }
func (r *AutoRootOp) String() string {
	return fmt.Sprintf("auto root %q flags %#x", r.Host, r.Flags)
}

// IsAutoRootBindable returns whether a dir entry name is selected for AutoRoot.
func IsAutoRootBindable(msg Msg, name string) bool {
	switch name {
	case "proc", "dev", "tmp", "mnt", "etc":

	case "": // guard against accidentally binding /
		// should be unreachable
		msg.Verbose("got unexpected root entry")

	default:
		return true
	}
	return false
}
