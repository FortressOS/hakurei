package system

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
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
func (t *Tmpfile) apply(*I) error {
	msg.Verbose("copying", t)

	if t.payload == nil {
		// this is a misuse of the API; do not return an error message
		return errors.New("invalid payload")
	}

	if b, err := os.Stat(t.src); err != nil {
		return newOpError("tmpfile", err, false)
	} else {
		if b.IsDir() {
			return newOpError("tmpfile", &os.PathError{Op: "stat", Path: t.src, Err: syscall.EISDIR}, false)
		}
		if s := b.Size(); s > t.n {
			return newOpError("tmpfile", &os.PathError{Op: "stat", Path: t.src, Err: syscall.ENOMEM}, false)
		}
	}

	if f, err := os.Open(t.src); err != nil {
		return newOpError("tmpfile", err, false)
	} else if _, err = io.CopyN(t.buf, f, t.n); err != nil {
		return newOpError("tmpfile", err, false)
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
