// Package seccomp provides high level wrappers around libseccomp.
package seccomp

import (
	"os"
	"runtime"
	"sync"
)

type exporter struct {
	presets FilterPreset
	flags   PrepareFlag
	r, w    *os.File

	prepareOnce sync.Once
	prepareErr  error
	closeOnce   sync.Once
	closeErr    error
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
		go func(fd uintptr) {
			ec <- preparePreset(int(fd), e.presets, e.flags)
			close(ec)
			_ = e.closeWrite()
			runtime.KeepAlive(e.w)
		}(e.w.Fd())
		e.exportErr = ec
		runtime.SetFinalizer(e, (*exporter).closeWrite)
	})
	return e.prepareErr
}

func (e *exporter) closeWrite() error {
	e.closeOnce.Do(func() {
		if e.w == nil {
			panic("closeWrite called on invalid exporter")
		}
		e.closeErr = e.w.Close()

		// no need for a finalizer anymore
		runtime.SetFinalizer(e, nil)
	})

	return e.closeErr
}

func newExporter(presets FilterPreset, flags PrepareFlag) *exporter {
	return &exporter{presets: presets, flags: flags}
}
