// Package helper runs external helpers with optional sandboxing.
package helper

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"hakurei.app/helper/proc"
)

var WaitDelay = 2 * time.Second

const (
	// HakureiHelper is set to 1 when args fd is enabled and 0 otherwise.
	HakureiHelper = "HAKUREI_HELPER"
	// HakureiStatus is set to 1 when stat fd is enabled and 0 otherwise.
	HakureiStatus = "HAKUREI_STATUS"
)

type Helper interface {
	// Start starts the helper process.
	Start() error
	// Wait blocks until Helper exits.
	Wait() error

	fmt.Stringer
}

func newHelperFiles(
	ctx context.Context,
	wt io.WriterTo,
	stat bool,
	argF func(argsFd, statFd int) []string,
	extraFiles []*os.File,
) (hl *helperFiles, args []string) {
	hl = new(helperFiles)
	hl.ctx = ctx
	hl.useArgsFd = wt != nil
	hl.useStatFd = stat

	hl.extraFiles = new(proc.ExtraFilesPre)
	for _, f := range extraFiles {
		_, v := hl.extraFiles.Append()
		*v = f
	}

	argsFd := -1
	if hl.useArgsFd {
		f := proc.NewWriterTo(wt)
		argsFd = int(proc.InitFile(f, hl.extraFiles))
		hl.files = append(hl.files, f)
	}

	statFd := -1
	if hl.useStatFd {
		f := proc.NewStat(&hl.stat)
		statFd = int(proc.InitFile(f, hl.extraFiles))
		hl.files = append(hl.files, f)
	}

	args = argF(argsFd, statFd)
	return
}

// helperFiles provides a generic wrapper around helper ipc.
type helperFiles struct {
	// whether argsFd is present
	useArgsFd bool
	// whether statFd is present
	useStatFd bool

	// closes statFd
	stat io.Closer
	// deferred extraFiles fulfillment
	files []proc.File
	// passed through to [proc.Fulfill] and [proc.InitFile]
	extraFiles *proc.ExtraFilesPre

	ctx context.Context
}
