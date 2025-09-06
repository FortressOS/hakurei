package system

import (
	"errors"
	"fmt"
	"os"
)

// Ensure ensures the existence of a directory.
func (sys *I) Ensure(name string, perm os.FileMode) *I {
	sys.ops = append(sys.ops, &mkdirOp{User, name, perm, false})
	return sys
}

// Ephemeral ensures the existence of a directory until its [Enablement] is no longer satisfied.
func (sys *I) Ephemeral(et Enablement, name string, perm os.FileMode) *I {
	sys.ops = append(sys.ops, &mkdirOp{et, name, perm, true})
	return sys
}

// mkdirOp implements [I.Ensure] and [I.Ephemeral].
type mkdirOp struct {
	et        Enablement
	path      string
	perm      os.FileMode
	ephemeral bool
}

func (m *mkdirOp) Type() Enablement { return m.et }

func (m *mkdirOp) apply(*I) error {
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

func (m *mkdirOp) revert(_ *I, ec *Criteria) error {
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

func (m *mkdirOp) Is(o Op) bool {
	target, ok := o.(*mkdirOp)
	return ok && m != nil && target != nil && *m == *target
}

func (m *mkdirOp) Path() string { return m.path }

func (m *mkdirOp) String() string {
	t := "ensure"
	if m.ephemeral {
		t = TypeString(m.Type())
	}
	return fmt.Sprintf("mode: %s type: %s path: %q", m.perm.String(), t, m.path)
}
