package system

import (
	"errors"
	"fmt"
	"os"

	"git.ophivana.moe/security/fortify/internal/fmsg"
)

// Ensure the existence and mode of a directory.
func (sys *I) Ensure(name string, perm os.FileMode) *I {
	sys.lock.Lock()
	defer sys.lock.Unlock()

	sys.ops = append(sys.ops, &Mkdir{User, name, perm, false})

	return sys
}

// Ephemeral ensures the temporary existence and mode of a directory through the life of et.
func (sys *I) Ephemeral(et Enablement, name string, perm os.FileMode) *I {
	sys.lock.Lock()
	defer sys.lock.Unlock()

	sys.ops = append(sys.ops, &Mkdir{et, name, perm, true})

	return sys
}

type Mkdir struct {
	et        Enablement
	path      string
	perm      os.FileMode
	ephemeral bool
}

func (m *Mkdir) Type() Enablement {
	return m.et
}

func (m *Mkdir) apply(_ *I) error {
	fmsg.VPrintln("ensuring directory", m)

	// create directory
	err := os.Mkdir(m.path, m.perm)
	if !errors.Is(err, os.ErrExist) {
		return fmsg.WrapErrorSuffix(err,
			fmt.Sprintf("cannot create directory %q:", m.path))
	}

	// directory exists, ensure mode
	return fmsg.WrapErrorSuffix(os.Chmod(m.path, m.perm),
		fmt.Sprintf("cannot change mode of %q to %s:", m.path, m.perm))
}

func (m *Mkdir) revert(_ *I, ec *Criteria) error {
	if !m.ephemeral {
		// skip non-ephemeral dir and do not log anything
		return nil
	}

	if ec.hasType(m) {
		fmsg.VPrintln("destroying ephemeral directory", m)
		return fmsg.WrapErrorSuffix(os.Remove(m.path),
			fmt.Sprintf("cannot remove ephemeral directory %q:", m.path))
	} else {
		fmsg.VPrintln("skipping ephemeral directory", m)
		return nil
	}
}

func (m *Mkdir) Is(o Op) bool {
	m0, ok := o.(*Mkdir)
	return ok && m0 != nil && *m == *m0
}

func (m *Mkdir) Path() string {
	return m.path
}

func (m *Mkdir) String() string {
	t := "Ensure"
	if m.ephemeral {
		t = TypeString(m.Type())
	}
	return fmt.Sprintf("mode: %s type: %s path: %q", m.perm.String(), t, m.path)
}
