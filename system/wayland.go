package system

import (
	"errors"
	"fmt"
	"os"

	"hakurei.app/system/acl"
	"hakurei.app/system/wayland"
)

// Wayland appends [WaylandOp] to [I].
func (sys *I) Wayland(syncFd **os.File, dst, src, appID, instanceID string) *I {
	sys.ops = append(sys.ops, &WaylandOp{syncFd, dst, src, appID, instanceID, wayland.Conn{}})
	return sys
}

// WaylandOp maintains a wayland socket with security-context-v1 attached via [wayland].
// The socket stops accepting connections once the pipe referred to by sync is closed.
// The socket is pathname only and is destroyed on revert.
type WaylandOp struct {
	sync              **os.File
	dst, src          string
	appID, instanceID string

	conn wayland.Conn
}

func (w *WaylandOp) Type() Enablement { return Process }

func (w *WaylandOp) apply(sys *I) error {
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

func (w *WaylandOp) revert(_ *I, ec *Criteria) error {
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

func (w *WaylandOp) Is(o Op) bool {
	target, ok := o.(*WaylandOp)
	return ok && w != nil && target != nil &&
		w.dst == target.dst && w.src == target.src &&
		w.appID == target.appID && w.instanceID == target.instanceID
}

func (w *WaylandOp) Path() string   { return w.dst }
func (w *WaylandOp) String() string { return fmt.Sprintf("wayland socket at %q", w.dst) }
