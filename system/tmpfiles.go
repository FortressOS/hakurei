package system

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
)

// CopyFile reads up to n bytes from src and writes the resulting byte slice to payloadP.
func (sys *I) CopyFile(payloadP *[]byte, src string, cap int, n int64) *I {
	buf := new(bytes.Buffer)
	buf.Grow(cap)
	sys.ops = append(sys.ops, &tmpfileOp{payloadP, src, n, buf})
	return sys
}

// tmpfileOp implements [I.CopyFile].
type tmpfileOp struct {
	payload *[]byte
	src     string

	n   int64
	buf *bytes.Buffer
}

func (t *tmpfileOp) Type() Enablement { return Process }

func (t *tmpfileOp) apply(sys *I) error {
	if t.payload == nil {
		// this is a misuse of the API; do not return a wrapped error
		return errors.New("invalid payload")
	}

	sys.verbose("copying", t)

	if b, err := sys.stat(t.src); err != nil {
		return newOpError("tmpfile", err, false)
	} else {
		if b.IsDir() {
			return newOpError("tmpfile", &os.PathError{Op: "stat", Path: t.src, Err: syscall.EISDIR}, false)
		}
		if s := b.Size(); s > t.n {
			return newOpError("tmpfile", &os.PathError{Op: "stat", Path: t.src, Err: syscall.ENOMEM}, false)
		}
	}

	var r io.ReadCloser
	if f, err := sys.open(t.src); err != nil {
		return newOpError("tmpfile", err, false)
	} else {
		r = f
	}
	if n, err := io.CopyN(t.buf, r, t.n); err != nil {
		if !errors.Is(err, io.EOF) {
			_ = r.Close()
			return newOpError("tmpfile", err, false)
		}
		sys.verbosef("copied %d bytes from %q", n, t.src)
	}
	if err := r.Close(); err != nil {
		return newOpError("tmpfile", err, false)
	}

	*t.payload = t.buf.Bytes()
	return nil
}
func (t *tmpfileOp) revert(*I, *Criteria) error { t.buf.Reset(); return nil }

func (t *tmpfileOp) Is(o Op) bool {
	target, ok := o.(*tmpfileOp)
	return ok && t != nil && target != nil &&
		t.src == target.src && t.n == target.n
}
func (t *tmpfileOp) Path() string   { return t.src }
func (t *tmpfileOp) String() string { return fmt.Sprintf("up to %d bytes from %q", t.n, t.src) }
