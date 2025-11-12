// Deprecated: This package will be removed in 0.4.
package proc

import (
	"context"
	"io"
	"os"
	"os/exec"
	"time"
	_ "unsafe" // for go:linkname

	"hakurei.app/internal/helper/proc"
)

//go:linkname FulfillmentTimeout hakurei.app/internal/helper/proc.FulfillmentTimeout
var FulfillmentTimeout time.Duration

// A File is an extra file with deferred initialisation.
type File = proc.File

// ExtraFilesPre is a linked list storing addresses of [os.File].
type ExtraFilesPre = proc.ExtraFilesPre

// Fulfill calls the [File.Fulfill] method on all files, starts cmd and blocks until all fulfillment completes.
//
//go:linkname Fulfill hakurei.app/internal/helper/proc.Fulfill
func Fulfill(ctx context.Context,
	v *[]*os.File, start func() error,
	files []File, extraFiles *ExtraFilesPre,
) (err error)

// InitFile initialises f as part of the slice extraFiles points to,
// and returns its final fd value.
//
//go:linkname InitFile hakurei.app/internal/helper/proc.InitFile
func InitFile(f File, extraFiles *ExtraFilesPre) (fd uintptr)

// BaseFile implements the Init method of the File interface and provides indirect access to extra file state.
type BaseFile = proc.BaseFile

//go:linkname ExtraFile hakurei.app/internal/helper/proc.ExtraFile
func ExtraFile(cmd *exec.Cmd, f *os.File) (fd uintptr)

//go:linkname ExtraFileSlice hakurei.app/internal/helper/proc.ExtraFileSlice
func ExtraFileSlice(extraFiles *[]*os.File, f *os.File) (fd uintptr)

// NewWriterTo returns a [File] that receives content from wt on fulfillment.
//
//go:linkname NewWriterTo hakurei.app/internal/helper/proc.NewWriterTo
func NewWriterTo(wt io.WriterTo) File

// NewStat returns a [File] implementing the behaviour
// of the receiving end of xdg-dbus-proxy stat fd.
//
//go:linkname NewStat hakurei.app/internal/helper/proc.NewStat
func NewStat(s *io.Closer) File

var (
	//go:linkname ErrStatFault hakurei.app/internal/helper/proc.ErrStatFault
	ErrStatFault error
	//go:linkname ErrStatRead hakurei.app/internal/helper/proc.ErrStatRead
	ErrStatRead error
)
