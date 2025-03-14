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
	// SetStdin sets the standard input of Helper.
	SetStdin(r io.Reader) Helper
	// SetStdout sets the standard output of Helper.
	SetStdout(w io.Writer) Helper
	// SetStderr sets the standard error of Helper.
	SetStderr(w io.Writer) Helper
	// SetEnv sets the environment of Helper.
	SetEnv(env []string) Helper

	// Start starts the helper process.
	// A status pipe is passed to the helper if stat is true.
	Start(stat bool) error
	// Wait blocks until Helper exits and releases all its resources.
	Wait() error

	fmt.Stringer
}

func newHelperCmd(
	h Helper, ctx context.Context, name string,
	wt io.WriterTo, argF func(argsFd, statFd int) []string,
	extraFiles []*os.File,
) (cmd *helperCmd) {
	cmd = new(helperCmd)
	cmd.r = h
	cmd.ctx = ctx

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
	// ref to parent
	r Helper

	// returns an array of arguments passed directly
	// to the helper process
	argF func(statFd int) []string
	// whether argsFd is present
	hasArgsFd bool

	// closes statFd
	stat io.Closer
	// deferred extraFiles fulfillment
	files []proc.File
	// passed through to [proc.Fulfill] and [proc.InitFile]
	extraFiles *proc.ExtraFilesPre

	ctx context.Context
	*exec.Cmd
}

func (h *helperCmd) SetStdin(r io.Reader) Helper  { h.Stdin = r; return h.r }
func (h *helperCmd) SetStdout(w io.Writer) Helper { h.Stdout = w; return h.r }
func (h *helperCmd) SetStderr(w io.Writer) Helper { h.Stderr = w; return h.r }
func (h *helperCmd) SetEnv(env []string) Helper   { h.Env = env; return h.r }

// finalise sets up the underlying [exec.Cmd] object.
func (h *helperCmd) finalise(stat bool) (args []string) {
	h.Env = slices.Grow(h.Env, 2)
	if h.hasArgsFd {
		h.Cmd.Env = append(h.Env, FortifyHelper+"=1")
	} else {
		h.Cmd.Env = append(h.Env, FortifyHelper+"=0")
	}

	statFd := -1
	if stat {
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
