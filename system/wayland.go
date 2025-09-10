package system

import (
	"errors"
	"fmt"
	"os"

	"hakurei.app/system/acl"
	"hakurei.app/system/wayland"
)

type waylandConn interface {
	Attach(p string) (err error)
	Bind(pathname, appID, instanceID string) (*os.File, error)
	Close() error
}

// Wayland maintains a wayland socket with security-context-v1 attached via [wayland].
// The socket stops accepting connections once the pipe referred to by sync is closed.
// The socket is pathname only and is destroyed on revert.
func (sys *I) Wayland(syncFd **os.File, dst, src, appID, instanceID string) *I {
	sys.ops = append(sys.ops, &waylandOp{syncFd, dst, src, appID, instanceID, new(wayland.Conn)})
	return sys
}

// waylandOp implements [I.Wayland].
type waylandOp struct {
	sync              **os.File
	dst, src          string
	appID, instanceID string

	conn waylandConn
}

func (w *waylandOp) Type() Enablement { return Process }

func (w *waylandOp) apply(sys *I) error {
	if w.sync == nil {
		// this is a misuse of the API; do not return a wrapped error
		return errors.New("invalid sync")
	}

	if err := w.conn.Attach(w.src); err != nil {
		return newOpError("wayland", err, false)
	} else {
		sys.verbosef("wayland attached on %q", w.src)
	}

	if sp, err := w.conn.Bind(w.dst, w.appID, w.instanceID); err != nil {
		return newOpError("wayland", err, false)
	} else {
		*w.sync = sp
		sys.verbosef("wayland listening on %q", w.dst)
		if err = sys.chmod(w.dst, 0); err != nil {
			return newOpError("wayland", err, false)
		}
		return newOpError("wayland", sys.aclUpdate(w.dst, sys.uid, acl.Read, acl.Write, acl.Execute), false)
	}
}

func (w *waylandOp) revert(sys *I, _ *Criteria) error {
	sys.verbosef("removing wayland socket on %q", w.dst)
	if err := sys.remove(w.dst); err != nil && !errors.Is(err, os.ErrNotExist) {
		return newOpError("wayland", err, true)
	}

	sys.verbosef("detaching from wayland on %q", w.src)
	return newOpError("wayland", w.conn.Close(), true)
}

func (w *waylandOp) Is(o Op) bool {
	target, ok := o.(*waylandOp)
	return ok && w != nil && target != nil &&
		w.dst == target.dst && w.src == target.src &&
		w.appID == target.appID && w.instanceID == target.instanceID
}

func (w *waylandOp) Path() string   { return w.dst }
func (w *waylandOp) String() string { return fmt.Sprintf("wayland socket at %q", w.dst) }
