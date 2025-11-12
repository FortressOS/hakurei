package system

import (
	"errors"
	"fmt"
	"os"

	"hakurei.app/container/check"
	"hakurei.app/hst"
	"hakurei.app/internal/system/acl"
	"hakurei.app/internal/system/wayland"
)

type waylandConn interface {
	Attach(p string) (err error)
	Bind(pathname, appID, instanceID string) (*os.File, error)
	Close() error
}

// Wayland maintains a wayland socket with security-context-v1 attached via [wayland].
// The socket stops accepting connections once the pipe referred to by sync is closed.
// The socket is pathname only and is destroyed on revert.
func (sys *I) Wayland(dst, src *check.Absolute, appID, instanceID string) *I {
	sys.ops = append(sys.ops, &waylandOp{nil,
		dst.String(), src.String(),
		appID, instanceID,
		new(wayland.Conn)})
	return sys
}

// waylandOp implements [I.Wayland].
type waylandOp struct {
	sync              *os.File
	dst, src          string
	appID, instanceID string

	conn waylandConn
}

func (w *waylandOp) Type() hst.Enablement { return Process }

func (w *waylandOp) apply(sys *I) error {
	if err := w.conn.Attach(w.src); err != nil {
		return newOpError("wayland", err, false)
	} else {
		sys.msg.Verbosef("wayland attached on %q", w.src)
	}

	if sp, err := w.conn.Bind(w.dst, w.appID, w.instanceID); err != nil {
		return newOpError("wayland", err, false)
	} else {
		w.sync = sp
		sys.msg.Verbosef("wayland listening on %q", w.dst)
		if err = sys.chmod(w.dst, 0); err != nil {
			return newOpError("wayland", err, false)
		}
		return newOpError("wayland", sys.aclUpdate(w.dst, sys.uid, acl.Read, acl.Write, acl.Execute), false)
	}
}

func (w *waylandOp) revert(sys *I, _ *Criteria) error {
	var (
		hangupErr error
		closeErr  error
		removeErr error
	)

	sys.msg.Verbosef("detaching from wayland on %q", w.src)
	if w.sync != nil {
		hangupErr = w.sync.Close()
	}
	closeErr = w.conn.Close()

	sys.msg.Verbosef("removing wayland socket on %q", w.dst)
	if err := sys.remove(w.dst); err != nil && !errors.Is(err, os.ErrNotExist) {
		removeErr = err
	}

	return newOpError("wayland", errors.Join(hangupErr, closeErr, removeErr), true)
}

func (w *waylandOp) Is(o Op) bool {
	target, ok := o.(*waylandOp)
	return ok && w != nil && target != nil &&
		w.dst == target.dst && w.src == target.src &&
		w.appID == target.appID && w.instanceID == target.instanceID
}

func (w *waylandOp) Path() string   { return w.dst }
func (w *waylandOp) String() string { return fmt.Sprintf("wayland socket at %q", w.dst) }
