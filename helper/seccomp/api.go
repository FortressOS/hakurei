package seccomp

import (
	"errors"
	"io"
	"os"
	"syscall"
)

func Export(opts SyscallOpts) (f *os.File, err error) {
	if f, err = tmpfile(); err != nil {
		return
	}
	if err = exportFilter(f.Fd(), opts); err != nil {
		return
	}
	_, err = f.Seek(0, io.SeekStart)
	return
}

/*
An Encoder writes a BPF program to an output stream.

Methods of Encoder are not safe for concurrent use.

An Encoder must not be copied after first use.
*/
type Encoder struct {
	*exporter
}

func (e *Encoder) Read(p []byte) (n int, err error) {
	if err = e.prepare(); err != nil {
		return
	}
	return e.r.Read(p)
}

func (e *Encoder) Close() error {
	if e.r == nil {
		return syscall.EINVAL
	}

	// this hangs if the cgo thread fails to exit
	return errors.Join(e.closeWrite(), <-e.exportErr)
}

// New returns an inactive Encoder instance.
func New(opts SyscallOpts) *Encoder {
	return &Encoder{newExporter(opts)}
}
