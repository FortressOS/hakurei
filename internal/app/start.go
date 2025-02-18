package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/helper"
	"git.gensokyo.uk/security/fortify/internal/app/shim"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
	"git.gensokyo.uk/security/fortify/internal/state"
	"git.gensokyo.uk/security/fortify/system"
)

const shimSetupTimeout = 5 * time.Second

func (a *app) Run(ctx context.Context, rs *fst.RunState) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	if rs == nil {
		panic("attempted to pass nil state to run")
	}

	// resolve exec paths
	shimExec := [2]string{helper.BubblewrapName}
	if len(a.seal.command) > 0 {
		shimExec[1] = a.seal.command[0]
	}
	for i, n := range shimExec {
		if len(n) == 0 {
			continue
		}
		if filepath.Base(n) == n {
			if s, err := exec.LookPath(n); err == nil {
				shimExec[i] = s
			} else {
				return fmsg.WrapError(err,
					fmt.Sprintf("executable file %q not found in $PATH", n))
			}
		}
	}

	// startup will go ahead, commit system setup
	if err := a.seal.sys.Commit(ctx); err != nil {
		return err
	}
	a.seal.sys.needRevert = true

	// start shim via manager
	a.shim = new(shim.Shim)
	waitErr := make(chan error, 1)
	if startTime, err := a.shim.Start(
		a.seal.sys.user.as,
		a.seal.sys.user.supp,
		a.seal.sys.sp,
	); err != nil {
		return err
	} else {
		// shim process created
		rs.Start = true

		shimSetupCtx, shimSetupCancel := context.WithDeadline(ctx, time.Now().Add(shimSetupTimeout))
		defer shimSetupCancel()

		// start waiting for shim
		go func() {
			waitErr <- a.shim.Unwrap().Wait()
			// cancel shim setup in case shim died before receiving payload
			shimSetupCancel()
		}()

		// send payload
		if err = a.shim.Serve(shimSetupCtx, &shim.Payload{
			Argv:  a.seal.command,
			Exec:  shimExec,
			Bwrap: a.seal.sys.bwrap,
			Home:  a.seal.sys.user.data,

			Verbose: fmsg.Load(),
		}); err != nil {
			return err
		}

		// shim accepted setup payload, create process state
		sd := state.State{
			ID:   *a.id,
			PID:  a.shim.Unwrap().Process.Pid,
			Time: *startTime,
		}

		// register process state
		var err0 = new(StateStoreError)
		err0.Inner, err0.DoErr = a.seal.store.Do(a.seal.sys.user.aid, func(c state.Cursor) {
			err0.InnerErr = c.Save(&sd, a.seal.ct)
		})
		a.seal.sys.saveState = true
		if err = err0.equiv("cannot save process state:"); err != nil {
			return err
		}
	}

	select {
	// wait for process and resolve exit code
	case err := <-waitErr:
		if err != nil {
			var exitError *exec.ExitError
			if !errors.As(err, &exitError) {
				// should be unreachable
				rs.WaitErr = err
			}

			// store non-zero return code
			rs.ExitCode = exitError.ExitCode()
		} else {
			rs.ExitCode = a.shim.Unwrap().ProcessState.ExitCode()
		}
		if fmsg.Load() {
			fmsg.Verbosef("process %d exited with exit code %d", a.shim.Unwrap().Process.Pid, rs.ExitCode)
		}

	// this is reached when a fault makes an already running shim impossible to continue execution
	// however a kill signal could not be delivered (should actually always happen like that since fsu)
	// the effects of this is similar to the alternative exit path and ensures shim death
	case err := <-a.shim.WaitFallback():
		rs.ExitCode = 255
		log.Printf("cannot terminate shim on faulted setup: %v", err)

	// alternative exit path relying on shim behaviour on monitor process exit
	case <-ctx.Done():
		fmsg.Verbose("alternative exit path selected")
	}

	// child process exited, resume output
	fmsg.Resume()

	// print queued up dbus messages
	if a.seal.dbusMsg != nil {
		a.seal.dbusMsg()
	}

	// update store and revert app setup transaction
	e := new(StateStoreError)
	e.Inner, e.DoErr = a.seal.store.Do(a.seal.sys.user.aid, func(b state.Cursor) {
		e.InnerErr = func() error {
			// destroy defunct state entry
			if cmd := a.shim.Unwrap(); cmd != nil && a.seal.sys.saveState {
				if err := b.Destroy(*a.id); err != nil {
					return err
				}
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
					fmsg.Verbose("no other launchers active, will clean up globals")
					ec.Set(system.User)
				} else {
					fmsg.Verbosef("found %d active launchers, cleaning up without globals", l)
				}

				// accumulate capabilities of other launchers
				for i, s := range states {
					if s.Config != nil {
						*rt |= s.Config.Confinement.Enablements
					} else {
						log.Printf("state entry %d does not contain config", i)
					}
				}
			}
			// invert accumulated enablements for cleanup
			for i := system.Enablement(0); i < system.Enablement(system.ELen); i++ {
				if !rt.Has(i) {
					ec.Set(i)
				}
			}
			if fmsg.Load() {
				labels := make([]string, 0, system.ELen+1)
				for i := system.Enablement(0); i < system.Enablement(system.ELen+2); i++ {
					if ec.Has(i) {
						labels = append(labels, system.TypeString(i))
					}
				}
				if len(labels) > 0 {
					fmsg.Verbose("reverting operations labelled", strings.Join(labels, ", "))
				}
			}

			if a.seal.sys.needRevert {
				if err := a.seal.sys.Revert(ec); err != nil {
					return err.(RevertCompoundError)
				}
			}

			return nil
		}()
	})

	e.Err = a.seal.store.Close()
	return e.equiv("error returned during cleanup:", e)
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
	if e.Inner && e.DoErr == nil && e.InnerErr == nil && e.Err == nil {
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
