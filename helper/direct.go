package helper

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"slices"
	"sync"
	"syscall"

	"git.gensokyo.uk/security/fortify/helper/proc"
)

// NewDirect initialises a new direct Helper instance with wt as the null-terminated argument writer.
// Function argF returns an array of arguments passed directly to the child process.
func NewDirect(
	ctx context.Context,
	name string,
	wt io.WriterTo,
	stat bool,
	argF func(argsFd, statFd int) []string,
	cmdF func(cmd *exec.Cmd),
) Helper {
	d := new(direct)
	d.helperCmd = newHelperCmd(ctx, name, wt, argF, stat, nil)
	if cmdF != nil {
		cmdF(d.helperCmd.Cmd)
	}
	return d
}

// direct starts the helper directly and manages status and args fd.
type direct struct {
	lock sync.RWMutex
	*helperCmd
}

func (h *direct) Start() error {
	h.lock.Lock()
	defer h.lock.Unlock()

	// Check for doubled Start calls before we defer failure cleanup. If the prior
	// call to Start succeeded, we don't want to spuriously close its pipes.
	if h.Cmd != nil && h.Cmd.Process != nil {
		return errors.New("exec: already started")
	}

	args := h.finalise()
	h.Cmd.Args = append(h.Cmd.Args, args...)
	return proc.Fulfill(h.ctx, &h.ExtraFiles, h.Cmd.Start, h.files, h.extraFiles)
}

func newHelperCmd(
	ctx context.Context,
	name string,
	wt io.WriterTo,
	argF func(argsFd, statFd int) []string,
	stat bool,
	extraFiles []*os.File,
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
