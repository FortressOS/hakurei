package system

import (
	"errors"
	"fmt"
	"os"

	"hakurei.app/container/check"
	"hakurei.app/hst"
	"hakurei.app/internal/acl"
	"hakurei.app/internal/wayland"
)

// Wayland maintains a wayland socket with security-context-v1 attached via [wayland].
// The socket stops accepting connections once the pipe referred to by sync is closed.
// The socket is pathname only and is destroyed on revert.
func (sys *I) Wayland(dst, src *check.Absolute, appID, instanceID string) *I {
	sys.ops = append(sys.ops, &waylandOp{nil,
		dst, src, appID, instanceID})
	return sys
}

// waylandOp implements [I.Wayland].
type waylandOp struct {
	ctx               *wayland.SecurityContext
	dst, src          *check.Absolute
	appID, instanceID string
}

func (w *waylandOp) Type() hst.Enablement { return Process }

func (w *waylandOp) apply(sys *I) (err error) {
	if w.ctx, err = sys.waylandNew(w.src, w.dst, w.appID, w.instanceID); err != nil {
		return newOpError("wayland", err, false)
	} else {
		sys.msg.Verbosef("wayland pathname socket on %q via %q", w.dst, w.src)

		if err = sys.chmod(w.dst.String(), 0); err != nil {
			if closeErr := w.ctx.Close(); closeErr != nil {
				return newOpError("wayland", errors.Join(err, closeErr), false)
			}
			return newOpError("wayland", err, false)
		}

		if err = sys.aclUpdate(w.dst.String(), sys.uid, acl.Read, acl.Write, acl.Execute); err != nil {
			if closeErr := w.ctx.Close(); closeErr != nil {
				return newOpError("wayland", errors.Join(err, closeErr), false)
			}
			return newOpError("wayland", err, false)
		}

		return nil
	}
}

func (w *waylandOp) revert(sys *I, _ *Criteria) error {
	var (
		hangupErr error
		removeErr error
	)

	sys.msg.Verbosef("hanging up wayland socket on %q", w.dst)
	if w.ctx != nil {
		hangupErr = w.ctx.Close()
	}
	if err := sys.remove(w.dst.String()); err != nil && !errors.Is(err, os.ErrNotExist) {
		removeErr = err
	}

	return newOpError("wayland", errors.Join(hangupErr, removeErr), true)
}

func (w *waylandOp) Is(o Op) bool {
	target, ok := o.(*waylandOp)
	return ok && w != nil && target != nil &&
		w.dst.Is(target.dst) && w.src.Is(target.src) &&
		w.appID == target.appID && w.instanceID == target.instanceID
}

func (w *waylandOp) Path() string   { return w.dst.String() }
func (w *waylandOp) String() string { return fmt.Sprintf("wayland socket at %q", w.dst) }
