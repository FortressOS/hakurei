package app

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"git.ophivana.moe/cat/fortify/helper"
	"git.ophivana.moe/cat/fortify/internal/fmsg"
	"git.ophivana.moe/cat/fortify/internal/shim"
	"git.ophivana.moe/cat/fortify/internal/state"
	"git.ophivana.moe/cat/fortify/internal/system"
	"git.ophivana.moe/cat/fortify/internal/verbose"
)

// Start starts the fortified child
func (a *app) Start() error {
	a.lock.Lock()
	defer a.lock.Unlock()

	// resolve exec paths
	shimExec := [3]string{a.seal.sys.executable, helper.BubblewrapName}
	if len(a.seal.command) > 0 {
		shimExec[2] = a.seal.command[0]
	}
	for i, n := range shimExec {
		if len(n) == 0 {
			continue
		}
		if filepath.Base(n) == n {
			if s, err := exec.LookPath(n); err == nil {
				shimExec[i] = s
			} else {
				return fmsg.WrapErrorSuffix(err,
					fmt.Sprintf("cannot find %q:", n))
			}
		}
	}

	if err := a.seal.sys.Commit(); err != nil {
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

	if wls, err := shim.ServeConfig(confSockPath, a.seal.sys.UID(), &shim.Payload{
		Argv:  a.seal.command,
		Exec:  shimExec,
		Bwrap: a.seal.sys.bwrap,
		WL:    a.seal.wlDone != nil,

		Verbose: verbose.Get(),
	}, a.seal.wl, a.seal.wlDone); err != nil {
		return fmsg.WrapErrorSuffix(err,
			"cannot listen on shim socket:")
	} else {
		a.wayland = wls
	}

	// start shim
	verbose.Println("starting shim as target user:", a.cmd)
	if err := a.cmd.Start(); err != nil {
		return fmsg.WrapErrorSuffix(err,
			"cannot start process:")
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
	return err.equiv("cannot save process state:")
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
		return fmsg.WrapErrorSuffix(e, a...)
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
			rt, ec := new(system.Enablements), new(system.Criteria)
			ec.Enablements = new(system.Enablements)
			ec.Set(system.Process)
			if states, err := b.Load(); err != nil {
				return err
			} else {
				if l := len(states); l == 0 {
					// cleanup globals as the final launcher
					verbose.Println("no other launchers active, will clean up globals")
					ec.Set(system.User)
				} else {
					verbose.Printf("found %d active launchers, cleaning up without globals\n", l)
				}

				// accumulate capabilities of other launchers
				for _, s := range states {
					*rt |= s.Capability
				}
			}
			// invert accumulated enablements for cleanup
			for i := system.Enablement(0); i < system.Enablement(system.ELen); i++ {
				if !rt.Has(i) {
					ec.Set(i)
				}
			}
			if verbose.Get() {
				labels := make([]string, 0, system.ELen+1)
				for i := system.Enablement(0); i < system.Enablement(system.ELen+2); i++ {
					if ec.Has(i) {
						labels = append(labels, system.TypeString(i))
					}
				}
				if len(labels) > 0 {
					verbose.Println("reverting operations labelled", strings.Join(labels, ", "))
				}
			}

			if err := a.seal.sys.Revert(ec); err != nil {
				return err.(RevertCompoundError)
			}

			return nil
		}()
	})

	e.Err = a.seal.store.Close()
	return r, e.equiv("error returned during cleanup:", e)
}
