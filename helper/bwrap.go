package helper

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"sync"
	"syscall"

	"git.gensokyo.uk/security/fortify/helper/bwrap"
	"git.gensokyo.uk/security/fortify/helper/proc"
)

// BubblewrapName is the file name or path to bubblewrap.
var BubblewrapName = "bwrap"

type bubblewrap struct {
	// final args fd of bwrap process
	argsFd uintptr

	// name of the command to run in bwrap
	name string

	lock sync.RWMutex
	*helperCmd
}

func (b *bubblewrap) Start() error {
	b.lock.Lock()
	defer b.lock.Unlock()

	// Check for doubled Start calls before we defer failure cleanup. If the prior
	// call to Start succeeded, we don't want to spuriously close its pipes.
	if b.Cmd != nil && b.Cmd.Process != nil {
		return errors.New("exec: already started")
	}

	args := b.finalise()
	b.Cmd.Args = slices.Grow(b.Cmd.Args, 4+len(args))
	b.Cmd.Args = append(b.Cmd.Args, "--args", strconv.Itoa(int(b.argsFd)), "--", b.name)
	b.Cmd.Args = append(b.Cmd.Args, args...)
	return proc.Fulfill(b.ctx, &b.ExtraFiles, b.Cmd.Start, b.files, b.extraFiles)
}

// MustNewBwrap initialises a new Bwrap instance with wt as the null-terminated argument writer.
// If wt is nil, the child process spawned by bwrap will not get an argument pipe.
// Function argF returns an array of arguments passed directly to the child process.
func MustNewBwrap(
	ctx context.Context,
	conf *bwrap.Config,
	name string,
	setpgid bool,
	wt io.WriterTo,
	argF func(argsFD, statFD int) []string,
	cmdF func(cmd *exec.Cmd),
	extraFiles []*os.File,
	syncFd *os.File,
	stat bool,
) Helper {
	b, err := NewBwrap(ctx, conf, name, setpgid, wt, argF, cmdF, extraFiles, syncFd, stat)
	if err != nil {
		panic(err.Error())
	} else {
		return b
	}
}

// NewBwrap initialises a new Bwrap instance with wt as the null-terminated argument writer.
// If wt is nil, the child process spawned by bwrap will not get an argument pipe.
// Function argF returns an array of arguments passed directly to the child process.
func NewBwrap(
	ctx context.Context,
	conf *bwrap.Config,
	name string,
	setpgid bool,
	wt io.WriterTo,
	argF func(argsFd, statFd int) []string,
	cmdF func(cmd *exec.Cmd),
	extraFiles []*os.File,
	syncFd *os.File,
	stat bool,
) (Helper, error) {
	b := new(bubblewrap)

	b.name = name
	b.helperCmd = newHelperCmd(ctx, BubblewrapName, wt, argF, extraFiles, stat)
	if setpgid {
		b.Cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}
	if cmdF != nil {
		cmdF(b.helperCmd.Cmd)
	}

	if v, err := NewCheckedArgs(conf.Args(syncFd, b.extraFiles, &b.files)); err != nil {
		return nil, err
	} else {
		f := proc.NewWriterTo(v)
		b.argsFd = proc.InitFile(f, b.extraFiles)
		b.files = append(b.files, f)
	}

	return b, nil
}
