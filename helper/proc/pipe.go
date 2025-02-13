package proc

import (
	"context"
	"errors"
	"io"
	"os"
)

// NewWriterTo returns a [File] that receives content from wt on fulfillment.
func NewWriterTo(wt io.WriterTo) File { return &writeToFile{wt: wt} }

// writeToFile exports the read end of a pipe with data written by an [io.WriterTo].
type writeToFile struct {
	wt io.WriterTo
	BaseFile
}

func (f *writeToFile) ErrCount() int { return 3 }
func (f *writeToFile) Fulfill(ctx context.Context, dispatchErr func(error)) error {
	r, w, err := os.Pipe()
	if err != nil {
		return err
	}
	f.Set(r)

	done := make(chan struct{})
	go func() { _, err = f.wt.WriteTo(w); dispatchErr(err); dispatchErr(w.Close()); close(done) }()
	go func() {
		select {
		case <-done:
			dispatchErr(nil)
		case <-ctx.Done():
			dispatchErr(w.Close()) // this aborts WriteTo with file already closed
		}
	}()

	return nil
}

// NewStat returns a [File] implementing the behaviour
// of the receiving end of xdg-dbus-proxy stat fd.
func NewStat(s *io.Closer) File { return &statFile{s: s} }

var (
	ErrStatFault = errors.New("generic stat fd fault")
	ErrStatRead  = errors.New("unexpected stat behaviour")
)

// statFile implements xdg-dbus-proxy stat fd behaviour.
type statFile struct {
	s *io.Closer
	BaseFile
}

func (f *statFile) ErrCount() int { return 2 }
func (f *statFile) Fulfill(ctx context.Context, dispatchErr func(error)) error {
	r, w, err := os.Pipe()
	if err != nil {
		return err
	}
	f.Set(w)

	done := make(chan struct{})
	go func() {
		defer close(done)
		var n int

		n, err = r.Read(make([]byte, 1))
		switch n {
		case -1:
			if err == nil {
				err = ErrStatFault
			}
			dispatchErr(err)
		case 0:
			if err == nil {
				err = ErrStatRead
			}
			dispatchErr(err)
		case 1:
			dispatchErr(err)
		default:
			panic("unreachable")
		}
	}()

	go func() {
		select {
		case <-done:
			dispatchErr(nil)
		case <-ctx.Done():
			dispatchErr(r.Close()) // this aborts Read with file already closed
		}
	}()

	// this gets closed by the caller
	*f.s = r
	return nil
}
