package system

import (
	"errors"
	"fmt"
	"os"

	"git.gensokyo.uk/security/fortify/acl"
	"git.gensokyo.uk/security/fortify/wl"
)

// Wayland sets up a wayland socket with a security context attached.
func (sys *I) Wayland(syncFd **os.File, dst, src, appID, instanceID string) *I {
	sys.lock.Lock()
	defer sys.lock.Unlock()

	sys.ops = append(sys.ops, &Wayland{syncFd, dst, src, appID, instanceID, wl.Conn{}})

	return sys
}

type Wayland struct {
	sync              **os.File
	dst, src          string
	appID, instanceID string

	conn wl.Conn
}

func (w *Wayland) Type() Enablement { return Process }

func (w *Wayland) apply(sys *I) error {
	if w.sync == nil {
		// this is a misuse of the API; do not return an error message
		return errors.New("invalid sync")
	}

	// the Wayland op is not repeatable
	if *w.sync != nil {
		// this is a misuse of the API; do not return an error message
		return errors.New("attempted to attach multiple wayland sockets")
	}

	if err := w.conn.Attach(w.src); err != nil {
		// make console output less nasty
		if errors.Is(err, os.ErrNotExist) {
			err = os.ErrNotExist
		}
		return sys.wrapErrSuffix(err,
			fmt.Sprintf("cannot attach to wayland on %q:", w.src))
	} else {
		sys.printf("wayland attached on %q", w.src)
	}

	if sp, err := w.conn.Bind(w.dst, w.appID, w.instanceID); err != nil {
		return sys.wrapErrSuffix(err,
			fmt.Sprintf("cannot bind to socket on %q:", w.dst))
	} else {
		*w.sync = sp
		sys.printf("wayland listening on %q", w.dst)
		return sys.wrapErrSuffix(errors.Join(os.Chmod(w.dst, 0), acl.Update(w.dst, sys.uid, acl.Read, acl.Write, acl.Execute)),
			fmt.Sprintf("cannot chmod socket on %q:", w.dst))
	}
}

func (w *Wayland) revert(sys *I, ec *Criteria) error {
	if ec.hasType(w) {
		sys.printf("removing wayland socket on %q", w.dst)
		if err := os.Remove(w.dst); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}

		sys.printf("detaching from wayland on %q", w.src)
		return sys.wrapErrSuffix(w.conn.Close(),
			fmt.Sprintf("cannot detach from wayland on %q:", w.src))
	} else {
		sys.printf("skipping wayland cleanup on %q", w.dst)
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
