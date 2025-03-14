// Package helper runs external helpers with optional sandboxing.
package helper

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"syscall"
	"time"

	"git.gensokyo.uk/security/fortify/helper/proc"
)

var (
	WaitDelay = 2 * time.Second
)

const (
	// FortifyHelper is set to 1 when args fd is enabled and 0 otherwise.
	FortifyHelper = "FORTIFY_HELPER"
	// FortifyStatus is set to 1 when stat fd is enabled and 0 otherwise.
	FortifyStatus = "FORTIFY_STATUS"
)

type Helper interface {
	// Start starts the helper process.
	Start() error
	// Wait blocks until Helper exits.
	Wait() error

	fmt.Stringer
}

func newHelperCmd(
	ctx context.Context, name string,
	wt io.WriterTo, argF func(argsFd, statFd int) []string,
	extraFiles []*os.File, stat bool,
) (cmd *helperCmd) {
	cmd = new(helperCmd)
	cmd.ctx = ctx
	cmd.hasStatFd = stat

	cmd.Cmd = commandContext(ctx, name)
	cmd.Cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
	cmd.WaitDelay = WaitDelay

	cmd.extraFiles = new(proc.ExtraFilesPre)
	for _, f := range extraFiles {
		_, v := cmd.extraFiles.Append()
		*v = f
	}

	argsFd := -1
	if wt != nil {
		f := proc.NewWriterTo(wt)
		argsFd = int(proc.InitFile(f, cmd.extraFiles))
		cmd.files = append(cmd.files, f)
		cmd.hasArgsFd = true
	}
	cmd.argF = func(statFd int) []string { return argF(argsFd, statFd) }

	return
}

// helperCmd wraps Cmd and implements methods shared across all Helper implementations.
type helperCmd struct {
	// returns an array of arguments passed directly
	// to the helper process
	argF func(statFd int) []string
	// whether argsFd is present
	hasArgsFd bool
	// whether statFd is present
	hasStatFd bool

	// closes statFd
	stat io.Closer
	// deferred extraFiles fulfillment
	files []proc.File
	// passed through to [proc.Fulfill] and [proc.InitFile]
	extraFiles *proc.ExtraFilesPre

	ctx context.Context
	*exec.Cmd
}

// finalise sets up the underlying [exec.Cmd] object.
func (h *helperCmd) finalise() (args []string) {
	h.Env = slices.Grow(h.Env, 2)
	if h.hasArgsFd {
		h.Cmd.Env = append(h.Env, FortifyHelper+"=1")
	} else {
		h.Cmd.Env = append(h.Env, FortifyHelper+"=0")
	}

	statFd := -1
	if h.hasStatFd {
		f := proc.NewStat(&h.stat)
		statFd = int(proc.InitFile(f, h.extraFiles))
		h.files = append(h.files, f)
		h.Cmd.Env = append(h.Cmd.Env, FortifyStatus+"=1")

		// stat is populated on fulfill
		h.Cmd.Cancel = func() error { return h.stat.Close() }
	} else {
		h.Cmd.Env = append(h.Cmd.Env, FortifyStatus+"=0")
	}
	return h.argF(statFd)
}

var commandContext = exec.CommandContext
