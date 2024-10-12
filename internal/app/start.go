package app

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"git.ophivana.moe/cat/fortify/helper"
	"git.ophivana.moe/cat/fortify/internal/shim"
	"git.ophivana.moe/cat/fortify/internal/state"
	"git.ophivana.moe/cat/fortify/internal/verbose"
)

type (
	// ProcessError encapsulates errors returned by starting *exec.Cmd
	ProcessError BaseError
	// ShimError encapsulates errors returned by shim.ServeConfig.
	ShimError BaseError
)

// Start starts the fortified child
func (a *app) Start() error {
	a.lock.Lock()
	defer a.lock.Unlock()

	// resolve exec paths
	e := [2]string{helper.BubblewrapName}
	if len(a.seal.command) > 0 {
		e[1] = a.seal.command[0]
	}
	for i, n := range e {
		if len(n) == 0 {
			continue
		}
		if filepath.Base(n) == n {
			if s, err := exec.LookPath(n); err == nil {
				e[i] = s
			} else {
				return (*ProcessError)(wrapError(err, fmt.Sprintf("cannot find %q: %v", n, err)))
			}
		}
	}

	if err := a.seal.sys.commit(); err != nil {
		return err
	}

	// select command builder
	var commandBuilder func(shimEnv string) (args []string)
	switch a.seal.launchOption {
	case LaunchMethodSudo:
		commandBuilder = a.commandBuilderSudo
	case LaunchMethodMachineCtl:
		commandBuilder = a.commandBuilderMachineCtl
	default:
		panic("unreachable")
	}

	// configure child process
	confSockPath := path.Join(a.seal.share, "shim")
	a.cmd = exec.Command(a.seal.toolPath, commandBuilder(shim.EnvShim+"="+confSockPath)...)
	a.cmd.Env = []string{}
	a.cmd.Stdin = os.Stdin
	a.cmd.Stdout = os.Stdout
	a.cmd.Stderr = os.Stderr
	a.cmd.Dir = a.seal.RunDirPath

	if wls, err := shim.ServeConfig(confSockPath, &shim.Payload{
		Argv:  a.seal.command,
		Exec:  e,
		Bwrap: a.seal.sys.bwrap,
		WL:    a.seal.wlDone != nil,

		Verbose: verbose.Get(),
	}, a.seal.wl, a.seal.wlDone); err != nil {
		return (*ShimError)(wrapError(err, "cannot listen on shim socket:", err))
	} else {
		a.wayland = wls
	}

	// start shim
	verbose.Println("starting shim as target user:", a.cmd)
	if err := a.cmd.Start(); err != nil {
		return (*ProcessError)(wrapError(err, "cannot start process:", err))
	}
	startTime := time.Now().UTC()

	// create process state
	sd := state.State{
		PID:        a.cmd.Process.Pid,
		Command:    a.seal.command,
		Capability: a.seal.et,
		Launcher:   a.seal.toolPath,
		Argv:       a.cmd.Args,
		Time:       startTime,
	}

	// register process state
	var err = new(StateStoreError)
	err.Inner, err.DoErr = a.seal.store.Do(func(b state.Backend) {
		err.InnerErr = b.Save(&sd)
	})
	return err.equiv("cannot save process state:", e)
}

// StateStoreError is returned for a failed state save
type StateStoreError struct {
	// whether inner function was called
	Inner bool
	// error returned by state.Store Do method
	DoErr error
	// error returned by state.Backend Save method
	InnerErr error
	// any other errors needing to be tracked
	Err error
}

func (e *StateStoreError) equiv(a ...any) error {
	if e.Inner == true && e.DoErr == nil && e.InnerErr == nil && e.Err == nil {
		return nil
	} else {
		return wrapError(e, a...)
	}
}

func (e *StateStoreError) Error() string {
	if e.Inner && e.InnerErr != nil {
		return e.InnerErr.Error()
	}

	if e.DoErr != nil {
		return e.DoErr.Error()
	}

	if e.Err != nil {
		return e.Err.Error()
	}

	return "(nil)"
}

func (e *StateStoreError) Unwrap() (errs []error) {
	errs = make([]error, 0, 3)
	if e.DoErr != nil {
		errs = append(errs, e.DoErr)
	}
	if e.InnerErr != nil {
		errs = append(errs, e.InnerErr)
	}
	if e.Err != nil {
		errs = append(errs, e.Err)
	}
	return
}

type RevertCompoundError interface {
	Error() string
	Unwrap() []error
}

func (a *app) Wait() (int, error) {
	a.lock.Lock()
	defer a.lock.Unlock()

	var r int

	// wait for process and resolve exit code
	if err := a.cmd.Wait(); err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) {
			// should be unreachable
			a.wait = err
		}

		// store non-zero return code
		r = exitError.ExitCode()
	} else {
		r = a.cmd.ProcessState.ExitCode()
	}

	verbose.Println("process", strconv.Itoa(a.cmd.Process.Pid), "exited with exit code", r)

	// close wayland connection
	if a.wayland != nil {
		close(a.seal.wlDone)
		if err := a.wayland.Close(); err != nil {
			fmt.Println("fortify: cannot close wayland connection:", err)
		}
	}

	// update store and revert app setup transaction
	e := new(StateStoreError)
	e.Inner, e.DoErr = a.seal.store.Do(func(b state.Backend) {
		e.InnerErr = func() error {
			// destroy defunct state entry
			if err := b.Destroy(a.cmd.Process.Pid); err != nil {
				return err
			}

			// enablements of remaining launchers
			rt, tags := new(state.Enablements), new(state.Enablements)
			tags.Set(state.EnableLength + 1)
			if states, err := b.Load(); err != nil {
				return err
			} else {
				if l := len(states); l == 0 {
					// cleanup globals as the final launcher
					verbose.Println("no other launchers active, will clean up globals")
					tags.Set(state.EnableLength)
				} else {
					verbose.Printf("found %d active launchers, cleaning up without globals\n", l)
				}

				// accumulate capabilities of other launchers
				for _, s := range states {
					*rt |= s.Capability
				}
			}
			// invert accumulated enablements for cleanup
			for i := state.Enablement(0); i < state.EnableLength; i++ {
				if !rt.Has(i) {
					tags.Set(i)
				}
			}
			if verbose.Get() {
				ct := make([]state.Enablement, 0, state.EnableLength)
				for i := state.Enablement(0); i < state.EnableLength; i++ {
					if tags.Has(i) {
						ct = append(ct, i)
					}
				}
				if len(ct) > 0 {
					verbose.Println("will revert operations tagged", ct, "as no remaining launchers hold these enablements")
				}
			}

			if err := a.seal.sys.revert(tags); err != nil {
				return err.(RevertCompoundError)
			}

			return nil
		}()
	})

	e.Err = a.seal.store.Close()
	return r, e.equiv("error returned during cleanup:", e)
}
