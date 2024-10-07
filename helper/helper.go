/*
Package helper runs external helpers and manages their status and args FDs.
*/
package helper

import (
	"errors"
	"io"
	"os/exec"
	"sync"
)

var (
	ErrStatusFault = errors.New("generic status pipe fault")
	ErrStatusRead  = errors.New("unexpected status response")
)

const (
	// FortifyHelper is set for the process launched by Helper.
	FortifyHelper = "FORTIFY_HELPER"
	// FortifyStatus is 1 when sync fd is enabled and 0 otherwise.
	FortifyStatus = "FORTIFY_STATUS"
)

// Helper wraps *exec.Cmd and manages status and args fd.
// Args is always 3 and status if set is always 4.
type Helper struct {
	p *pipes

	argF func(argsFD, statFD int) []string
	*exec.Cmd

	lock sync.RWMutex
}

func (h *Helper) StartNotify(ready chan error) error {
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

func (h *Helper) Wait() error {
	h.lock.RLock()
	defer h.lock.RUnlock()

	if h.Cmd.Process == nil {
		return errors.New("exec: not started")
	}
	if h.Cmd.ProcessState != nil {
		return errors.New("exec: Wait was already called")
	}

	defer h.p.mustClosePipes()
	return h.Cmd.Wait()
}

func (h *Helper) Close() error {
	return h.p.closeStatus()
}

func (h *Helper) Start() error {
	return h.StartNotify(nil)
}

var execCommand = exec.Command

// New initialises a new Helper instance with wt as the null-terminated argument writer.
// Function argF returns an array of arguments passed directly to the child process.
func New(wt io.WriterTo, name string, argF func(argsFD, statFD int) []string) *Helper {
	if wt == nil {
		panic("attempted to create helper with invalid argument writer")
	}

	return &Helper{p: &pipes{args: wt}, argF: argF, Cmd: execCommand(name)}
}
