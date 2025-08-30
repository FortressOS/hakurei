package system

import (
	"errors"
	"fmt"
	"os"

	"hakurei.app/system/acl"
	"hakurei.app/system/wayland"
)

// Wayland sets up a wayland socket with a security context attached.
func (sys *I) Wayland(syncFd **os.File, dst, src, appID, instanceID string) *I {
	sys.lock.Lock()
	defer sys.lock.Unlock()

	sys.ops = append(sys.ops, &Wayland{syncFd, dst, src, appID, instanceID, wayland.Conn{}})

	return sys
}

type Wayland struct {
	sync              **os.File
	dst, src          string
	appID, instanceID string

	conn wayland.Conn
}

func (w *Wayland) Type() Enablement { return Process }

func (w *Wayland) apply(sys *I) error {
	if w.sync == nil {
		// this is a misuse of the API; do not return a wrapped error
		return errors.New("invalid sync")
	}

	// the Wayland op is not repeatable
	if *w.sync != nil {
		// this is a misuse of the API; do not return a wrapped error
		return errors.New("attempted to attach multiple wayland sockets")
	}

	if err := w.conn.Attach(w.src); err != nil {
		return newOpError("wayland", err, false)
	} else {
		msg.Verbosef("wayland attached on %q", w.src)
	}

	if sp, err := w.conn.Bind(w.dst, w.appID, w.instanceID); err != nil {
		return newOpError("wayland", err, false)
	} else {
		*w.sync = sp
		msg.Verbosef("wayland listening on %q", w.dst)
		if err = os.Chmod(w.dst, 0); err != nil {
			return newOpError("wayland", err, false)
		}
		return newOpError("wayland", acl.Update(w.dst, sys.uid, acl.Read, acl.Write, acl.Execute), false)
	}
}

func (w *Wayland) revert(_ *I, ec *Criteria) error {
	if ec.hasType(w) {
		msg.Verbosef("removing wayland socket on %q", w.dst)
		if err := os.Remove(w.dst); err != nil && !errors.Is(err, os.ErrNotExist) {
			return newOpError("wayland", err, true)
		}

		msg.Verbosef("detaching from wayland on %q", w.src)
		return newOpError("wayland", w.conn.Close(), true)
	} else {
		msg.Verbosef("skipping wayland cleanup on %q", w.dst)
		return nil
	}
}

func (w *Wayland) Is(o Op) bool {
	w0, ok := o.(*Wayland)
	return ok && w.dst == w0.dst && w.src == w0.src &&
		w.appID == w0.appID && w.instanceID == w0.instanceID
}

func (w *Wayland) Path() string   { return w.dst }
func (w *Wayland) String() string { return fmt.Sprintf("wayland socket at %q", w.dst) }
