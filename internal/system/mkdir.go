package system

import (
	"errors"
	"fmt"
	"os"

	"git.ophivana.moe/cat/fortify/internal/fmsg"
	"git.ophivana.moe/cat/fortify/internal/state"
	"git.ophivana.moe/cat/fortify/internal/verbose"
)

// Ensure the existence and mode of a directory.
func (sys *I) Ensure(name string, perm os.FileMode) {
	sys.lock.Lock()
	defer sys.lock.Unlock()

	sys.ops = append(sys.ops, &Mkdir{User, name, perm, false})
}

// Ephemeral ensures the temporary existence and mode of a directory through the life of et.
func (sys *I) Ephemeral(et state.Enablement, name string, perm os.FileMode) {
	sys.lock.Lock()
	defer sys.lock.Unlock()

	sys.ops = append(sys.ops, &Mkdir{et, name, perm, true})
}

type Mkdir struct {
	et        state.Enablement
	path      string
	perm      os.FileMode
	ephemeral bool
}

func (m *Mkdir) Type() state.Enablement {
	return m.et
}

func (m *Mkdir) apply(_ *I) error {
	verbose.Println("ensuring directory", m)

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
		verbose.Println("destroying ephemeral directory", m)
		return fmsg.WrapErrorSuffix(os.Remove(m.path),
			fmt.Sprintf("cannot remove ephemeral directory %q:", m.path))
	} else {
		verbose.Println("skipping ephemeral directory", m)
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
	return fmt.Sprintf("mode: %s path: %q", m.perm.String(), m.path)
}
