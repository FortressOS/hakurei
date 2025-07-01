package seccomp

import (
	"context"
	"errors"
	"syscall"

	"git.gensokyo.uk/security/hakurei/helper/proc"
)

const (
	PresetStrict = PresetExt | PresetDenyNS | PresetDenyTTY | PresetDenyDevel
)

// New returns an inactive Encoder instance.
func New(rules []NativeRule, flags PrepareFlag) *Encoder { return &Encoder{newExporter(rules, flags)} }

// Load loads a filter into the kernel.
func Load(rules []NativeRule, flags PrepareFlag) error { return Prepare(-1, rules, flags) }

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

// NewFile returns an instance of exporter implementing [proc.File].
func NewFile(rules []NativeRule, flags PrepareFlag) proc.File {
	return &File{rules: rules, flags: flags}
}

// File implements [proc.File] and provides access to the read end of exporter pipe.
type File struct {
	rules []NativeRule
	flags PrepareFlag
	proc.BaseFile
}

func (f *File) ErrCount() int { return 2 }
func (f *File) Fulfill(ctx context.Context, dispatchErr func(error)) error {
	e := newExporter(f.rules, f.flags)
	if err := e.prepare(); err != nil {
		return err
	}
	f.Set(e.r)
	go func() {
		select {
		case err := <-e.exportErr:
			dispatchErr(nil)
			dispatchErr(err)
		case <-ctx.Done():
			dispatchErr(e.closeWrite())
			dispatchErr(<-e.exportErr)
		}
	}()
	return nil
}
