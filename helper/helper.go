// Package helper runs external helpers with optional sandboxing and manages their status/args pipes.
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
	// Stdin sets the standard input of Helper.
	Stdin(r io.Reader) Helper
	// Stdout sets the standard output of Helper.
	Stdout(w io.Writer) Helper
	// Stderr sets the standard error of Helper.
	Stderr(w io.Writer) Helper
	// SetEnv sets the environment of Helper.
	SetEnv(env []string) Helper

	// Start starts the helper process.
	// A status pipe is passed to the helper if stat is true.
	Start(ctx context.Context, stat bool) error
	// Wait blocks until Helper exits and releases all its resources.
	Wait() error

	fmt.Stringer
}

func newHelperCmd(
	h Helper, name string,
	wt io.WriterTo, argF func(argsFd, statFd int) []string,
	extraFiles []*os.File,
) (cmd *helperCmd) {
	cmd = new(helperCmd)

	cmd.r = h
	cmd.name = name

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

	name           string
	stdin          io.Reader
	stdout, stderr io.Writer
	env            []string
	*exec.Cmd
}

func (h *helperCmd) Stdin(r io.Reader) Helper   { h.stdin = r; return h.r }
func (h *helperCmd) Stdout(w io.Writer) Helper  { h.stdout = w; return h.r }
func (h *helperCmd) Stderr(w io.Writer) Helper  { h.stderr = w; return h.r }
func (h *helperCmd) SetEnv(env []string) Helper { h.env = env; return h.r }

// finalise initialises the underlying [exec.Cmd] object.
func (h *helperCmd) finalise(ctx context.Context, stat bool) (args []string) {
	h.Cmd = commandContext(ctx, h.name)
	h.Cmd.Stdin, h.Cmd.Stdout, h.Cmd.Stderr = h.stdin, h.stdout, h.stderr
	h.Cmd.Env = slices.Grow(h.env, 2)
	if h.hasArgsFd {
		h.Cmd.Env = append(h.Cmd.Env, FortifyHelper+"=1")
	} else {
		h.Cmd.Env = append(h.Cmd.Env, FortifyHelper+"=0")
	}

	h.Cmd.Cancel = func() error { return h.Cmd.Process.Signal(syscall.SIGTERM) }
	h.Cmd.WaitDelay = WaitDelay

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
