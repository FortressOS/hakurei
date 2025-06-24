package helper

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"slices"
	"sync"

	"git.gensokyo.uk/security/hakurei/helper/proc"
	"git.gensokyo.uk/security/hakurei/sandbox"
)

// New initialises a Helper instance with wt as the null-terminated argument writer.
func New(
	ctx context.Context,
	name string,
	wt io.WriterTo,
	stat bool,
	argF func(argsFd, statFd int) []string,
	cmdF func(container *sandbox.Container),
	extraFiles []*os.File,
) Helper {
	var args []string
	h := new(helperContainer)
	h.helperFiles, args = newHelperFiles(ctx, wt, stat, argF, extraFiles)
	h.Container = sandbox.New(ctx, name, args...)
	h.WaitDelay = WaitDelay
	if cmdF != nil {
		cmdF(h.Container)
	}
	return h
}

// helperContainer provides a [sandbox.Container] wrapper around helper ipc.
type helperContainer struct {
	started bool

	mu sync.Mutex
	*helperFiles
	*sandbox.Container
}

func (h *helperContainer) Start() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.started {
		return errors.New("helper: already started")
	}
	h.started = true

	h.Env = slices.Grow(h.Env, 2)
	if h.useArgsFd {
		h.Env = append(h.Env, HakureiHelper+"=1")
	} else {
		h.Env = append(h.Env, HakureiHelper+"=0")
	}
	if h.useStatFd {
		h.Env = append(h.Env, HakureiStatus+"=1")

		// stat is populated on fulfill
		h.Cancel = func(*exec.Cmd) error { return h.stat.Close() }
	} else {
		h.Env = append(h.Env, HakureiStatus+"=0")
	}

	return proc.Fulfill(h.helperFiles.ctx, &h.ExtraFiles, func() error {
		if err := h.Container.Start(); err != nil {
			return err
		}
		return h.Container.Serve()
	}, h.files, h.extraFiles)
}
