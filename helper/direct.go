package helper

import (
	"context"
	"errors"
	"io"
	"sync"

	"git.gensokyo.uk/security/fortify/helper/proc"
)

// direct wraps *exec.Cmd and manages status and args fd.
// Args is always 3 and status if set is always 4.
type direct struct {
	lock sync.RWMutex
	*helperCmd
}

func (h *direct) Start(stat bool) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	// Check for doubled Start calls before we defer failure cleanup. If the prior
	// call to Start succeeded, we don't want to spuriously close its pipes.
	if h.Cmd != nil && h.Cmd.Process != nil {
		return errors.New("exec: already started")
	}

	args := h.finalise(stat)
	h.Cmd.Args = append(h.Cmd.Args, args...)
	return proc.Fulfill(h.ctx, h.Cmd, h.files, h.extraFiles)
}

// New initialises a new direct Helper instance with wt as the null-terminated argument writer.
// Function argF returns an array of arguments passed directly to the child process.
func New(ctx context.Context, wt io.WriterTo, name string, argF func(argsFd, statFd int) []string) Helper {
	d := new(direct)
	d.helperCmd = newHelperCmd(d, ctx, name, wt, argF, nil)
	return d
}
