package proc

import (
	"context"
	"os"
	"os/exec"
	"syscall"
	"time"
)

var FulfillmentTimeout = 2 * time.Second

// A File is an extra file with deferred initialisation.
type File interface {
	// Init initialises File state. Init must not be called more than once.
	Init(fd uintptr, v **os.File)
	// Fd returns the fd value set on initialisation.
	Fd() uintptr
	// ErrCount returns count of error values emitted during fulfillment.
	ErrCount() int
	// Fulfill is called prior to process creation and must populate its corresponding file address.
	// Error values sent to ec must match the return value of ErrCount.
	// Fulfill must not be called more than once.
	Fulfill(ctx context.Context, ec chan<- error) error
}

// Fulfill calls the [File.Fulfill] method on all files, starts cmd and blocks until all fulfillment completes.
func Fulfill(ctx context.Context, cmd *exec.Cmd, files []File) (err error) {
	var ecs int
	for _, o := range files {
		ecs += o.ErrCount()
	}
	ec := make(chan error, ecs)

	c, cancel := context.WithTimeout(ctx, FulfillmentTimeout)
	defer cancel()

	for _, o := range files {
		err = o.Fulfill(c, ec)
		if err != nil {
			return
		}
	}

	if err = cmd.Start(); err != nil {
		return
	}

	for ecs > 0 {
		select {
		case err = <-ec:
			ecs--
			if err != nil {
				break
			}
		case <-c.Done():
			err = syscall.ECANCELED
			break
		}
	}
	return
}

// InitFile initialises f as part of the slice extraFiles points to,
// and returns its final fd value.
func InitFile(f File, extraFiles *[]*os.File) (fd uintptr) {
	fd = ExtraFileSlice(extraFiles, nil)
	f.Init(fd, &(*extraFiles)[len(*extraFiles)-1])
	return
}

// BaseFile implements the Init method of the File interface and provides indirect access to extra file state.
type BaseFile struct {
	fd uintptr
	v  **os.File
}

func (f *BaseFile) Init(fd uintptr, v **os.File) {
	if v == nil || fd < 3 {
		panic("invalid extra file initial state")
	}
	if f.v != nil {
		panic("extra file initialised twice")
	}
	f.fd, f.v = fd, v
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

func ExtraFile(cmd *exec.Cmd, f *os.File) (fd uintptr) {
	return ExtraFileSlice(&cmd.ExtraFiles, f)
}

func ExtraFileSlice(extraFiles *[]*os.File, f *os.File) (fd uintptr) {
	// ExtraFiles: If non-nil, entry i becomes file descriptor 3+i.
	fd = uintptr(3 + len(*extraFiles))
	*extraFiles = append(*extraFiles, f)
	return
}
