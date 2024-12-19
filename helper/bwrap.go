package helper

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"strconv"
	"sync"

	"git.gensokyo.uk/security/fortify/helper/bwrap"
	"git.gensokyo.uk/security/fortify/internal/proc"
)

// BubblewrapName is the file name or path to bubblewrap.
var BubblewrapName = "bwrap"

type bubblewrap struct {
	// bwrap child file name
	name string

	// bwrap pipes
	p *pipes
	// sync pipe
	sync *os.File
	// returns an array of arguments passed directly
	// to the child process spawned by bwrap
	argF func(argsFD, statFD int) []string

	// pipes received by the child
	// nil if no pipes are required
	cp *pipes

	lock sync.RWMutex
	*exec.Cmd
}

func (b *bubblewrap) StartNotify(ready chan error) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if ready != nil && b.cp == nil {
		panic("attempted to start with status monitoring on a bwrap child initialised without pipes")
	}

	// Check for doubled Start calls before we defer failure cleanup. If the prior
	// call to Start succeeded, we don't want to spuriously close its pipes.
	if b.Cmd.Process != nil {
		return errors.New("exec: already started")
	}

	// prepare bwrap pipe and args
	if argsFD, _, err := b.p.prepareCmd(b.Cmd); err != nil {
		return err
	} else {
		b.Cmd.Args = append(b.Cmd.Args, "--args", strconv.Itoa(argsFD), "--", b.name)
	}

	// prepare child args and pipes if enabled
	if b.cp != nil {
		b.cp.ready = ready
		if argsFD, statFD, err := b.cp.prepareCmd(b.Cmd); err != nil {
			return err
		} else {
			b.Cmd.Args = append(b.Cmd.Args, b.argF(argsFD, statFD)...)
		}
	} else {
		b.Cmd.Args = append(b.Cmd.Args, b.argF(-1, -1)...)
	}

	if ready != nil {
		b.Cmd.Env = append(b.Cmd.Env, FortifyHelper+"=1", FortifyStatus+"=1")
	} else if b.cp != nil {
		b.Cmd.Env = append(b.Cmd.Env, FortifyHelper+"=1", FortifyStatus+"=0")
	} else {
		b.Cmd.Env = append(b.Cmd.Env, FortifyHelper+"=1", FortifyStatus+"=-1")
	}

	if b.sync != nil {
		b.Cmd.Args = append(b.Cmd.Args, "--sync-fd", strconv.Itoa(int(proc.ExtraFile(b.Cmd, b.sync))))
	}

	if err := b.Cmd.Start(); err != nil {
		return err
	}

	// write bwrap args first
	if err := b.p.readyWriteArgs(); err != nil {
		return err
	}

	// write child args if enabled
	if b.cp != nil {
		if err := b.cp.readyWriteArgs(); err != nil {
			return err
		}
	}

	return nil
}

func (b *bubblewrap) Close() error {
	if b.cp == nil {
		panic("attempted to close bwrap child initialised without pipes")
	}

	return b.cp.closeStatus()
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
func MustNewBwrap(conf *bwrap.Config, wt io.WriterTo, name string, argF func(argsFD, statFD int) []string) Helper {
	b, err := NewBwrap(conf, wt, name, argF)
	if err != nil {
		panic(err.Error())
	} else {
		return b
	}
}

// NewBwrap initialises a new Bwrap instance with wt as the null-terminated argument writer.
// If wt is nil, the child process spawned by bwrap will not get an argument pipe.
// Function argF returns an array of arguments passed directly to the child process.
func NewBwrap(conf *bwrap.Config, wt io.WriterTo, name string, argF func(argsFD, statFD int) []string) (Helper, error) {
	b := new(bubblewrap)

	if args, err := NewCheckedArgs(conf.Args()); err != nil {
		return nil, err
	} else {
		b.p = &pipes{args: args}
	}

	b.sync = conf.Sync()
	b.argF = argF
	b.name = name
	if wt != nil {
		b.cp = &pipes{args: wt}
	}
	b.Cmd = execCommand(BubblewrapName)

	return b, nil
}
