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
	extraFiles []*os.File,
) Helper {
	d, args := newHelperCmd(ctx, name, wt, stat, argF, extraFiles)
	d.Args = append(d.Args, args...)
	if cmdF != nil {
		cmdF(d.Cmd)
	}
	return d
}

func newHelperCmd(
	ctx context.Context,
	name string,
	wt io.WriterTo,
	stat bool,
	argF func(argsFd, statFd int) []string,
	extraFiles []*os.File,
) (cmd *helperCmd, args []string) {
	cmd = new(helperCmd)
	cmd.helperFiles, args = newHelperFiles(ctx, wt, stat, argF, extraFiles)
	cmd.Cmd = exec.CommandContext(ctx, name)
	cmd.Cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
	cmd.WaitDelay = WaitDelay
	return
}

// helperCmd provides a [exec.Cmd] wrapper around helper ipc.
type helperCmd struct {
	mu sync.RWMutex
	*helperFiles
	*exec.Cmd
}

func (h *helperCmd) Start() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Check for doubled Start calls before we defer failure cleanup. If the prior
	// call to Start succeeded, we don't want to spuriously close its pipes.
	if h.Cmd != nil && h.Cmd.Process != nil {
		return errors.New("helper: already started")
	}

	h.Env = slices.Grow(h.Env, 2)
	if h.useArgsFd {
		h.Env = append(h.Env, FortifyHelper+"=1")
	} else {
		h.Env = append(h.Env, FortifyHelper+"=0")
	}
	if h.useStatFd {
		h.Env = append(h.Env, FortifyStatus+"=1")

		// stat is populated on fulfill
		h.Cancel = func() error { return h.stat.Close() }
	} else {
		h.Env = append(h.Env, FortifyStatus+"=0")
	}

	return proc.Fulfill(h.helperFiles.ctx, &h.ExtraFiles, h.Cmd.Start, h.files, h.extraFiles)
}
