package system

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"git.gensokyo.uk/security/fortify/acl"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
)

// CopyFile registers an Op that copies path dst from src.
func (sys *I) CopyFile(dst, src string) *I {
	return sys.CopyFileType(Process, dst, src)
}

// CopyFileType registers a file copying Op labelled with type et.
func (sys *I) CopyFileType(et Enablement, dst, src string) *I {
	sys.lock.Lock()
	sys.ops = append(sys.ops, &Tmpfile{et, tmpfileCopy, dst, src})
	sys.lock.Unlock()

	sys.UpdatePermType(et, dst, acl.Read)

	return sys
}

const (
	tmpfileCopy uint8 = iota
)

type Tmpfile struct {
	et       Enablement
	method   uint8
	dst, src string
}

func (t *Tmpfile) Type() Enablement {
	return t.et
}

func (t *Tmpfile) apply(_ *I) error {
	switch t.method {
	case tmpfileCopy:
		fmsg.Verbose("publishing tmpfile", t)
		return fmsg.WrapErrorSuffix(copyFile(t.dst, t.src),
			fmt.Sprintf("cannot copy tmpfile %q:", t.dst))
	default:
		panic("invalid tmpfile method " + strconv.Itoa(int(t.method)))
	}
}

func (t *Tmpfile) revert(_ *I, ec *Criteria) error {
	if ec.hasType(t) {
		fmsg.Verbosef("removing tmpfile %q", t.dst)
		return fmsg.WrapErrorSuffix(os.Remove(t.dst),
			fmt.Sprintf("cannot remove tmpfile %q:", t.dst))
	} else {
		fmsg.Verbosef("skipping tmpfile %q", t.dst)
		return nil
	}
}

func (t *Tmpfile) Is(o Op) bool {
	t0, ok := o.(*Tmpfile)
	return ok && t0 != nil && *t == *t0
}

func (t *Tmpfile) Path() string { return t.src }

func (t *Tmpfile) String() string {
	switch t.method {
	case tmpfileCopy:
		return fmt.Sprintf("%q from %q", t.dst, t.src)
	default:
		panic("invalid tmpfile method " + strconv.Itoa(int(t.method)))
	}
}

func copyFile(dst, src string) error {
	dstD, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	srcD, err := os.Open(src)
	if err != nil {
		return errors.Join(err, dstD.Close())
	}

	_, err = io.Copy(dstD, srcD)
	return errors.Join(err, dstD.Close(), srcD.Close())
}
