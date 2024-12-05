package system

import (
	"errors"
	"fmt"
	"os"

	"git.ophivana.moe/security/fortify/acl"
	"git.ophivana.moe/security/fortify/internal/fmsg"
	"git.ophivana.moe/security/fortify/wl"
)

// Wayland sets up a wayland socket with a security context attached.
func (sys *I) Wayland(dst, src, appID, instanceID string) *I {
	sys.lock.Lock()
	defer sys.lock.Unlock()

	sys.ops = append(sys.ops, Wayland{[2]string{dst, src}, new(wl.Conn), appID, instanceID})

	return sys
}

type Wayland struct {
	pair [2]string
	conn *wl.Conn

	appID, instanceID string
}

func (w Wayland) Type() Enablement {
	return Process
}

func (w Wayland) apply(sys *I) error {
	if err := w.conn.Attach(w.pair[1]); err != nil {
		return fmsg.WrapErrorSuffix(err,
			fmt.Sprintf("cannot attach to wayland on %q:", w.pair[1]))
	} else {
		fmsg.VPrintf("wayland attached on %q", w.pair[1])
	}

	if sp, err := w.conn.Bind(w.pair[0], w.appID, w.instanceID); err != nil {
		return fmsg.WrapErrorSuffix(err,
			fmt.Sprintf("cannot bind to socket on %q:", w.pair[0]))
	} else {
		sys.sp = sp
		fmsg.VPrintf("wayland listening on %q", w.pair[0])
		return fmsg.WrapErrorSuffix(errors.Join(os.Chmod(w.pair[0], 0), acl.UpdatePerm(w.pair[0], sys.uid, acl.Read, acl.Write, acl.Execute)),
			fmt.Sprintf("cannot chmod socket on %q:", w.pair[0]))
	}
}

func (w Wayland) revert(_ *I, ec *Criteria) error {
	if ec.hasType(w) {
		fmsg.VPrintf("removing wayland socket on %q", w.pair[0])
		if err := os.Remove(w.pair[0]); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}

		fmsg.VPrintf("detaching from wayland on %q", w.pair[1])
		return fmsg.WrapErrorSuffix(w.conn.Close(),
			fmt.Sprintf("cannot detach from wayland on %q:", w.pair[1]))
	} else {
		fmsg.VPrintf("skipping wayland cleanup on %q", w.pair[0])
		return nil
	}
}

func (w Wayland) Is(o Op) bool {
	w0, ok := o.(Wayland)
	return ok && w.pair == w0.pair
}

func (w Wayland) Path() string {
	return w.pair[0]
}

func (w Wayland) String() string {
	return fmt.Sprintf("wayland socket at %q", w.pair[0])
}
