package proc

import (
	"context"
	"os"
	"os/exec"
	"sync/atomic"
	"syscall"
	"time"
)

var FulfillmentTimeout = 2 * time.Second

// A File is an extra file with deferred initialisation.
type File interface {
	// Init initialises File state. Init must not be called more than once.
	Init(fd uintptr, v **os.File) uintptr
	// Fd returns the fd value set on initialisation.
	Fd() uintptr
	// ErrCount returns count of error values emitted during fulfillment.
	ErrCount() int
	// Fulfill is called prior to process creation and must populate its corresponding file address.
	// Calls to dispatchErr must match the return value of ErrCount.
	// Fulfill must not be called more than once.
	Fulfill(ctx context.Context, dispatchErr func(error)) error
}

// ExtraFilesPre is a linked list storing addresses of [os.File].
type ExtraFilesPre struct {
	n *ExtraFilesPre
	v *os.File
}

// Append grows the list by one entry and returns an address of the address of [os.File] stored in the new entry.
func (f *ExtraFilesPre) Append() (uintptr, **os.File) { return f.append(3) }

// Files returns a slice pointing to a continuous segment of memory containing all addresses stored in f in order.
func (f *ExtraFilesPre) Files() []*os.File { return f.copy(make([]*os.File, 0, f.len())) }

func (f *ExtraFilesPre) append(i uintptr) (uintptr, **os.File) {
	if f.n == nil {
		f.n = new(ExtraFilesPre)
		return i, &f.v
	}
	return f.n.append(i + 1)
}
func (f *ExtraFilesPre) len() uintptr {
	if f == nil {
		return 0
	}
	return f.n.len() + 1
}
func (f *ExtraFilesPre) copy(e []*os.File) []*os.File {
	if f == nil {
		// the public methods ensure the first call is never nil;
		// the last element is unused, slice it off here
		return e[:len(e)-1]
	}
	return f.n.copy(append(e, f.v))
}

// Fulfill calls the [File.Fulfill] method on all files, starts cmd and blocks until all fulfillment completes.
func Fulfill(ctx context.Context,
	v *[]*os.File, start func() error,
	files []File, extraFiles *ExtraFilesPre,
) (err error) {
	var ecs int
	for _, o := range files {
		ecs += o.ErrCount()
	}
	ec := make(chan error, ecs)

	c, cancel := context.WithTimeout(ctx, FulfillmentTimeout)
	defer cancel()

	for _, f := range files {
		err = f.Fulfill(c, makeDispatchErr(f, ec))
		if err != nil {
			return
		}
	}

	*v = extraFiles.Files()
	if err = start(); err != nil {
		return
	}

	for ecs > 0 {
		select {
		case err = <-ec:
			ecs--
			if err != nil {
				break
			}
		case <-ctx.Done():
			err = syscall.ECANCELED
			break
		}
	}
	return
}

// InitFile initialises f as part of the slice extraFiles points to,
// and returns its final fd value.
func InitFile(f File, extraFiles *ExtraFilesPre) (fd uintptr) { return f.Init(extraFiles.Append()) }

// BaseFile implements the Init method of the File interface and provides indirect access to extra file state.
type BaseFile struct {
	fd uintptr
	v  **os.File
}

func (f *BaseFile) Init(fd uintptr, v **os.File) uintptr {
	if v == nil || fd < 3 {
		panic("invalid extra file initial state")
	}
	if f.v != nil {
		panic("extra file initialised twice")
	}
	f.fd, f.v = fd, v
	return fd
}

func (f *BaseFile) Fd() uintptr {
	if f.v == nil {
		panic("use of uninitialised extra file")
	}
	return f.fd
}

func (f *BaseFile) Set(v *os.File) {
	*f.v = v // runtime guards against use before init
}

func makeDispatchErr(f File, ec chan<- error) func(error) {
	c := new(atomic.Int32)
	c.Store(int32(f.ErrCount()))
	return func(err error) {
		if c.Add(-1) < 0 {
			panic("unexpected error dispatches")
		}
		ec <- err
	}
}

func ExtraFile(cmd *exec.Cmd, f *os.File) (fd uintptr) {
	return ExtraFileSlice(&cmd.ExtraFiles, f)
}

func ExtraFileSlice(extraFiles *[]*os.File, f *os.File) (fd uintptr) {
	// ExtraFiles: If non-nil, entry i becomes file descriptor 3+i.
	fd = uintptr(3 + len(*extraFiles))
	*extraFiles = append(*extraFiles, f)
	return
}
