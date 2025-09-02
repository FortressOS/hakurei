package system

import (
	"errors"
	"fmt"
	"os"
)

// Ensure appends [MkdirOp] to [I] with its [Enablement] ignored.
func (sys *I) Ensure(name string, perm os.FileMode) *I {
	sys.ops = append(sys.ops, &MkdirOp{User, name, perm, false})
	return sys
}

// Ephemeral appends an ephemeral [MkdirOp] to [I].
func (sys *I) Ephemeral(et Enablement, name string, perm os.FileMode) *I {
	sys.ops = append(sys.ops, &MkdirOp{et, name, perm, true})
	return sys
}

// MkdirOp ensures the existence of a directory.
// For ephemeral, the directory is destroyed once [Enablement] is no longer satisfied.
type MkdirOp struct {
	et        Enablement
	path      string
	perm      os.FileMode
	ephemeral bool
}

func (m *MkdirOp) Type() Enablement { return m.et }

func (m *MkdirOp) apply(*I) error {
	msg.Verbose("ensuring directory", m)

	// create directory
	if err := os.Mkdir(m.path, m.perm); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return newOpError("mkdir", err, false)
		}
		// directory exists, ensure mode
		return newOpError("mkdir", os.Chmod(m.path, m.perm), false)
	} else {
		return nil
	}
}

func (m *MkdirOp) revert(_ *I, ec *Criteria) error {
	if !m.ephemeral {
		// skip non-ephemeral dir and do not log anything
		return nil
	}

	if ec.hasType(m.Type()) {
		msg.Verbose("destroying ephemeral directory", m)
		return newOpError("mkdir", os.Remove(m.path), true)
	} else {
		msg.Verbose("skipping ephemeral directory", m)
		return nil
	}
}

func (m *MkdirOp) Is(o Op) bool {
	target, ok := o.(*MkdirOp)
	return ok && m != nil && target != nil && *m == *target
}

func (m *MkdirOp) Path() string { return m.path }

func (m *MkdirOp) String() string {
	t := "ensure"
	if m.ephemeral {
		t = TypeString(m.Type())
	}
	return fmt.Sprintf("mode: %s type: %s path: %q", m.perm.String(), t, m.path)
}
