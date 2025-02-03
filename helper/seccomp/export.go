package seccomp

import (
	"io/fs"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
)

type exporter struct {
	opts SyscallOpts
	r, w *os.File

	prepareOnce sync.Once
	prepareErr  error
	closeErr    atomic.Pointer[error]
	exportErr   <-chan error
}

func (e *exporter) prepare() error {
	e.prepareOnce.Do(func() {
		if r, w, err := os.Pipe(); err != nil {
			e.prepareErr = err
			return
		} else {
			e.r, e.w = r, w
		}

		ec := make(chan error, 1)
		go func() { ec <- exportFilter(e.w.Fd(), e.opts); close(ec); _ = e.closeWrite() }()
		e.exportErr = ec
		runtime.SetFinalizer(e, (*exporter).closeWrite)
	})
	return e.prepareErr
}

func (e *exporter) closeWrite() error {
	if !e.closeErr.CompareAndSwap(nil, &fs.ErrInvalid) {
		return *e.closeErr.Load()
	}
	if e.w == nil {
		return fs.ErrInvalid
	}
	err := e.w.Close()
	e.closeErr.Store(&err)

	// no need for a finalizer anymore
	runtime.SetFinalizer(e, nil)

	return err
}

func newExporter(opts SyscallOpts) *exporter {
	return &exporter{opts: opts}
}
