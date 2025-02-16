package system

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"syscall"

	"git.gensokyo.uk/security/fortify/internal/fmsg"
)

// CopyFile registers an Op that copies from src.
// A buffer is initialised with size cap and the Op faults if bytes read exceed n.
func (sys *I) CopyFile(payload *[]byte, src string, cap int, n int64) *I {
	buf := new(bytes.Buffer)
	buf.Grow(cap)

	sys.lock.Lock()
	sys.ops = append(sys.ops, &Tmpfile{payload, src, n, buf})
	sys.lock.Unlock()

	return sys
}

type Tmpfile struct {
	payload *[]byte
	src     string

	n   int64
	buf *bytes.Buffer
}

func (t *Tmpfile) Type() Enablement { return Process }
func (t *Tmpfile) apply(_ *I) error {
	fmsg.Verbose("copying", t)

	if b, err := os.Stat(t.src); err != nil {
		return fmsg.WrapErrorSuffix(err,
			fmt.Sprintf("cannot stat %q:", t.src))
	} else {
		if b.IsDir() {
			return fmsg.WrapErrorSuffix(syscall.EISDIR,
				fmt.Sprintf("%q is a directory", t.src))
		}
		if s := b.Size(); s > t.n {
			return fmsg.WrapErrorSuffix(syscall.ENOMEM,
				fmt.Sprintf("file %q is too long: %d > %d",
					t.src, s, t.n))
		}
	}

	if f, err := os.Open(t.src); err != nil {
		return fmsg.WrapErrorSuffix(err,
			fmt.Sprintf("cannot open %q:", t.src))
	} else if _, err = io.CopyN(t.buf, f, t.n); err != nil {
		return fmsg.WrapErrorSuffix(err,
			fmt.Sprintf("cannot read from %q:", t.src))
	}

	*t.payload = t.buf.Bytes()
	return nil
}
func (t *Tmpfile) revert(*I, *Criteria) error { t.buf.Reset(); return nil }

func (t *Tmpfile) Is(o Op) bool {
	t0, ok := o.(*Tmpfile)
	return ok && t0 != nil &&
		t.src == t0.src && t.n == t0.n
}
func (t *Tmpfile) Path() string   { return t.src }
func (t *Tmpfile) String() string { return fmt.Sprintf("up to %d bytes from %q", t.n, t.src) }
