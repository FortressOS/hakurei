package helper

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"strconv"
	"sync"

	"git.gensokyo.uk/security/fortify/helper/bwrap"
)

// BubblewrapName is the file name or path to bubblewrap.
var BubblewrapName = "bwrap"

type bubblewrap struct {
	// bwrap child file name
	name string

	// bwrap pipes
	control *pipes
	// returns an array of arguments passed directly
	// to the child process spawned by bwrap
	argF func(argsFD, statFD int) []string

	// pipes received by the child
	// nil if no pipes are required
	controlPt *pipes

	lock sync.RWMutex
	*exec.Cmd
}

func (b *bubblewrap) StartNotify(ready chan error) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if ready != nil && b.controlPt == nil {
		panic("attempted to start with status monitoring on a bwrap child initialised without pipes")
	}

	// Check for doubled Start calls before we defer failure cleanup. If the prior
	// call to Start succeeded, we don't want to spuriously close its pipes.
	if b.Cmd.Process != nil {
		return errors.New("exec: already started")
	}

	// prepare bwrap pipe and args
	if argsFD, _, err := b.control.prepareCmd(b.Cmd); err != nil {
		return err
	} else {
		b.Cmd.Args = append(b.Cmd.Args, "--args", strconv.Itoa(argsFD), "--", b.name)
	}

	// prepare child args and pipes if enabled
	if b.controlPt != nil {
		b.controlPt.ready = ready
		if argsFD, statFD, err := b.controlPt.prepareCmd(b.Cmd); err != nil {
			return err
		} else {
			b.Cmd.Args = append(b.Cmd.Args, b.argF(argsFD, statFD)...)
		}
	} else {
		b.Cmd.Args = append(b.Cmd.Args, b.argF(-1, -1)...)
	}

	if ready != nil {
		b.Cmd.Env = append(b.Cmd.Env, FortifyHelper+"=1", FortifyStatus+"=1")
	} else if b.controlPt != nil {
		b.Cmd.Env = append(b.Cmd.Env, FortifyHelper+"=1", FortifyStatus+"=0")
	} else {
		b.Cmd.Env = append(b.Cmd.Env, FortifyHelper+"=1", FortifyStatus+"=-1")
	}

	if err := b.Cmd.Start(); err != nil {
		return err
	}

	// write bwrap args first
	if err := b.control.readyWriteArgs(); err != nil {
		return err
	}

	// write child args if enabled
	if b.controlPt != nil {
		if err := b.controlPt.readyWriteArgs(); err != nil {
			return err
		}
	}

	return nil
}

func (b *bubblewrap) Close() error {
	if b.controlPt == nil {
		panic("attempted to close bwrap child initialised without pipes")
	}

	return b.controlPt.closeStatus()
}

func (b *bubblewrap) Start() error {
	return b.StartNotify(nil)
}

func (b *bubblewrap) Unwrap() *exec.Cmd {
	return b.Cmd
}

// MustNewBwrap initialises a new Bwrap instance with wt as the null-terminated argument writer.
// If wt is nil, the child process spawned by bwrap will not get an argument pipe.
// Function argF returns an array of arguments passed directly to the child process.
func MustNewBwrap(
	conf *bwrap.Config, name string,
	wt io.WriterTo, argF func(argsFD, statFD int) []string,
	extraFiles []*os.File,
	syncFd *os.File,
) Helper {
	b, err := NewBwrap(conf, name, wt, argF, extraFiles, syncFd)
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
	conf *bwrap.Config, name string,
	wt io.WriterTo, argF func(argsFD, statFD int) []string,
	extraFiles []*os.File,
	syncFd *os.File,
) (Helper, error) {
	b := new(bubblewrap)

	b.argF = argF
	b.name = name
	if wt != nil {
		b.controlPt = &pipes{args: wt}
	}

	b.Cmd = execCommand(BubblewrapName)
	b.control = new(pipes)
	args := conf.Args()
	if fdArgs, err := conf.FDArgs(syncFd, &extraFiles); err != nil {
		return nil, err
	} else if b.control.args, err = NewCheckedArgs(append(args, fdArgs...)); err != nil {
		return nil, err
	} else {
		b.Cmd.ExtraFiles = extraFiles
	}

	return b, nil
}
