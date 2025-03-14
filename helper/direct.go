package helper

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"sync"

	"git.gensokyo.uk/security/fortify/helper/proc"
)

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

// NewDirect initialises a new direct Helper instance with wt as the null-terminated argument writer.
// Function argF returns an array of arguments passed directly to the child process.
func NewDirect(
	ctx context.Context,
	wt io.WriterTo,
	name string,
	argF func(argsFd, statFd int) []string,
	cmdF func(cmd *exec.Cmd),
	stat bool,
) Helper {
	d := new(direct)
	d.helperCmd = newHelperCmd(ctx, name, wt, argF, nil, stat)
	if cmdF != nil {
		cmdF(d.helperCmd.Cmd)
	}
	return d
}
