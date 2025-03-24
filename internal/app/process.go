package app

import (
	"context"
	"errors"
	"log"
	"os/exec"
	"time"

	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/internal"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
	"git.gensokyo.uk/security/fortify/internal/state"
	"git.gensokyo.uk/security/fortify/system"
)

const shimSetupTimeout = 5 * time.Second

func (seal *outcome) Run(rs *fst.RunState) error {
	if !seal.f.CompareAndSwap(false, true) {
		// run does much more than just starting a process; calling it twice, even if the first call fails, will result
		// in inconsistent state that is impossible to clean up; return here to limit damage and hopefully give the
		// other Run a chance to return
		panic("attempted to run twice")
	}

	if rs == nil {
		panic("invalid state")
	}

	// read comp values early to allow for early failure
	fmsg.Verbosef("version %s", internal.Version())
	fmsg.Verbosef("setuid helper at %s", internal.MustFsuPath())

	/*
		prepare/revert os state
	*/

	if err := seal.sys.Commit(seal.ctx); err != nil {
		return err
	}
	store := state.NewMulti(seal.runDirPath)
	deferredStoreFunc := func(c state.Cursor) error { return nil }
	defer func() {
		var revertErr error
		storeErr := new(StateStoreError)
		storeErr.Inner, storeErr.DoErr = store.Do(seal.user.aid.unwrap(), func(c state.Cursor) {
			revertErr = func() error {
				storeErr.InnerErr = deferredStoreFunc(c)

				/*
					revert app setup transaction
				*/

				var rt system.Enablement
				ec := system.Process
				if states, err := c.Load(); err != nil {
					// revert per-process state here to limit damage
					storeErr.OpErr = err
					return seal.sys.Revert((*system.Criteria)(&ec))
				} else {
					if l := len(states); l == 0 {
						fmsg.Verbose("no other launchers active, will clean up globals")
						ec |= system.User
					} else {
						fmsg.Verbosef("found %d active launchers, cleaning up without globals", l)
					}

					// accumulate enablements of remaining launchers
					for i, s := range states {
						if s.Config != nil {
							rt |= s.Config.Confinement.Enablements
						} else {
							log.Printf("state entry %d does not contain config", i)
						}
					}
				}
				ec |= rt ^ (system.EWayland | system.EX11 | system.EDBus | system.EPulse)
				if fmsg.Load() {
					if ec > 0 {
						fmsg.Verbose("reverting operations type", system.TypeString(ec))
					}
				}

				return seal.sys.Revert((*system.Criteria)(&ec))
			}()
		})
		storeErr.save([]error{revertErr, store.Close()})
		rs.RevertErr = storeErr.equiv("error returned during cleanup:")
	}()

	/*
		shim process lifecycle
	*/

	waitErr := make(chan error, 1)
	cmd := new(shimProcess)
	if startTime, err := cmd.Start(
		seal.user.aid.String(),
		seal.user.supp,
	); err != nil {
		return err
	} else {
		// whether/when the fsu process was created
		rs.Time = startTime
	}

	ctx, cancel := context.WithTimeout(seal.ctx, shimSetupTimeout)
	defer cancel()

	go func() {
		waitErr <- cmd.Unwrap().Wait()
		// cancel shim setup in case shim died before receiving payload
		cancel()
	}()

	if err := cmd.Serve(ctx, &shimParams{
		Container: seal.container,
		Home:      seal.user.data,

		Verbose: fmsg.Load(),
	}); err != nil {
		return err
	}

	// shim accepted setup payload, create process state
	sd := state.State{
		ID:   seal.id.unwrap(),
		PID:  cmd.Unwrap().Process.Pid,
		Time: *rs.Time,
	}
	var earlyStoreErr = new(StateStoreError) // returned after blocking on waitErr
	earlyStoreErr.Inner, earlyStoreErr.DoErr = store.Do(seal.user.aid.unwrap(), func(c state.Cursor) {
		earlyStoreErr.InnerErr = c.Save(&sd, seal.ct)
	})
	// destroy defunct state entry
	deferredStoreFunc = func(c state.Cursor) error { return c.Destroy(seal.id.unwrap()) }

	select {
	case err := <-waitErr: // block until fsu/shim returns
		if err != nil {
			var exitError *exec.ExitError
			if !errors.As(err, &exitError) {
				// should be unreachable
				rs.WaitErr = err
			}

			// store non-zero return code
			rs.ExitCode = exitError.ExitCode()
		} else {
			rs.ExitCode = cmd.Unwrap().ProcessState.ExitCode()
		}
		if fmsg.Load() {
			fmsg.Verbosef("process %d exited with exit code %d", cmd.Unwrap().Process.Pid, rs.ExitCode)
		}

	// this is reached when a fault makes an already running shim impossible to continue execution
	// however a kill signal could not be delivered (should actually always happen like that since fsu)
	// the effects of this is similar to the alternative exit path and ensures shim death
	case err := <-cmd.Fallback():
		rs.ExitCode = 255
		log.Printf("cannot terminate shim on faulted setup: %v", err)

	// alternative exit path relying on shim behaviour on monitor process exit
	case <-seal.ctx.Done():
		fmsg.Verbose("alternative exit path selected")
	}

	fmsg.Resume()
	if seal.sync != nil {
		if err := seal.sync.Close(); err != nil {
			log.Printf("cannot close wayland security context: %v", err)
		}
	}
	if seal.dbusMsg != nil {
		seal.dbusMsg()
	}

	return earlyStoreErr.equiv("cannot save process state:")
}
