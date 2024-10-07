package helper

import (
	"errors"
	"io"
	"os/exec"
	"sync"
)

// direct wraps *exec.Cmd and manages status and args fd.
// Args is always 3 and status if set is always 4.
type direct struct {
	// helper pipes
	// cannot be nil
	p *pipes

	// returns an array of arguments passed directly
	// to the helper process
	argF func(argsFD, statFD int) []string

	lock sync.RWMutex
	*exec.Cmd
}

func (h *direct) StartNotify(ready chan error) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	// Check for doubled Start calls before we defer failure cleanup. If the prior
	// call to Start succeeded, we don't want to spuriously close its pipes.
	if h.Cmd.Process != nil {
		return errors.New("exec: already started")
	}

	h.p.ready = ready
	if argsFD, statFD, err := h.p.prepareCmd(h.Cmd); err != nil {
		return err
	} else {
		h.Cmd.Args = append(h.Cmd.Args, h.argF(argsFD, statFD)...)
	}

	if ready != nil {
		h.Cmd.Env = append(h.Cmd.Env, FortifyHelper+"=1", FortifyStatus+"=1")
	} else {
		h.Cmd.Env = append(h.Cmd.Env, FortifyHelper+"=1", FortifyStatus+"=0")
	}

	if err := h.Cmd.Start(); err != nil {
		return err
	}
	if err := h.p.readyWriteArgs(); err != nil {
		return err
	}

	return nil
}

func (h *direct) Wait() error {
	h.lock.RLock()
	defer h.lock.RUnlock()

	if h.Cmd.Process == nil {
		return errors.New("exec: not started")
	}
	defer h.p.mustClosePipes()
	if h.Cmd.ProcessState != nil {
		return errors.New("exec: Wait was already called")
	}

	return h.Cmd.Wait()
}

func (h *direct) Close() error {
	return h.p.closeStatus()
}

func (h *direct) Start() error {
	return h.StartNotify(nil)
}

func (h *direct) Unwrap() *exec.Cmd {
	return h.Cmd
}

// New initialises a new direct Helper instance with wt as the null-terminated argument writer.
// Function argF returns an array of arguments passed directly to the child process.
func New(wt io.WriterTo, name string, argF func(argsFD, statFD int) []string) Helper {
	if wt == nil {
		panic("attempted to create helper with invalid argument writer")
	}

	return &direct{p: &pipes{args: wt}, argF: argF, Cmd: execCommand(name)}
}
