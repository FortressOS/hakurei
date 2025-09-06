package system

import (
	"errors"
	"fmt"
	"os"

	"hakurei.app/system/acl"
	"hakurei.app/system/wayland"
)

// Wayland maintains a wayland socket with security-context-v1 attached via [wayland].
// The socket stops accepting connections once the pipe referred to by sync is closed.
// The socket is pathname only and is destroyed on revert.
func (sys *I) Wayland(syncFd **os.File, dst, src, appID, instanceID string) *I {
	sys.ops = append(sys.ops, &waylandOp{syncFd, dst, src, appID, instanceID, wayland.Conn{}})
	return sys
}

// waylandOp implements [I.Wayland].
type waylandOp struct {
	sync              **os.File
	dst, src          string
	appID, instanceID string

	conn wayland.Conn
}

func (w *waylandOp) Type() Enablement { return Process }

func (w *waylandOp) apply(sys *I) error {
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

func (w *waylandOp) revert(_ *I, ec *Criteria) error {
	if ec.hasType(w.Type()) {
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

func (w *waylandOp) Is(o Op) bool {
	target, ok := o.(*waylandOp)
	return ok && w != nil && target != nil &&
		w.dst == target.dst && w.src == target.src &&
		w.appID == target.appID && w.instanceID == target.instanceID
}

func (w *waylandOp) Path() string   { return w.dst }
func (w *waylandOp) String() string { return fmt.Sprintf("wayland socket at %q", w.dst) }
