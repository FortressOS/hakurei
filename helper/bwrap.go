package helper

import (
	"context"
	"io"
	"os"
	"os/exec"
	"slices"
	"strconv"

	"git.gensokyo.uk/security/fortify/helper/bwrap"
	"git.gensokyo.uk/security/fortify/helper/proc"
)

// BubblewrapName is the file name or path to bubblewrap.
var BubblewrapName = "bwrap"

// MustNewBwrap initialises a new Bwrap instance with wt as the null-terminated argument writer.
// If wt is nil, the child process spawned by bwrap will not get an argument pipe.
// Function argF returns an array of arguments passed directly to the child process.
func MustNewBwrap(
	ctx context.Context,
	name string,
	wt io.WriterTo,
	stat bool,
	argF func(argsFd, statFd int) []string,
	cmdF func(cmd *exec.Cmd),
	extraFiles []*os.File,
	conf *bwrap.Config,
	syncFd *os.File,
) Helper {
	b, err := NewBwrap(ctx, name, wt, stat, argF, cmdF, extraFiles, conf, syncFd)
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
	name string,
	wt io.WriterTo,
	stat bool,
	argF func(argsFd, statFd int) []string,
	cmdF func(cmd *exec.Cmd),
	extraFiles []*os.File,
	conf *bwrap.Config,
	syncFd *os.File,
) (Helper, error) {
	b, args := newHelperCmd(ctx, BubblewrapName, wt, stat, argF, extraFiles)

	var argsFd uintptr
	if v, err := NewCheckedArgs(conf.Args(syncFd, b.extraFiles, &b.files)); err != nil {
		return nil, err
	} else {
		f := proc.NewWriterTo(v)
		argsFd = proc.InitFile(f, b.extraFiles)
		b.files = append(b.files, f)
	}

	b.Args = slices.Grow(b.Args, 4+len(args))
	b.Args = append(b.Args, "--args", strconv.Itoa(int(argsFd)), "--", name)
	b.Args = append(b.Args, args...)
	if cmdF != nil {
		cmdF(b.Cmd)
	}
	return b, nil
}
