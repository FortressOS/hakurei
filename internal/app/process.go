package app

import (
	"context"
	"encoding/gob"
	"errors"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"hakurei.app/container"
	"hakurei.app/internal"
	"hakurei.app/internal/app/state"
	"hakurei.app/internal/hlog"
	"hakurei.app/system"
)

const shimWaitTimeout = 5 * time.Second

// RunState stores the outcome of a call to [Outcome.Run].
type RunState struct {
	// Time is the exact point in time where the process was created.
	// Location must be set to UTC.
	//
	// Time is nil if no process was ever created.
	Time *time.Time
	// RevertErr is stored by the deferred revert call.
	RevertErr error
	// WaitErr is the generic error value created by the standard library.
	WaitErr error

	syscall.WaitStatus
}

// setStart stores the current time in [RunState] once.
func (rs *RunState) setStart() {
	if rs.Time != nil {
		panic("attempted to store time twice")
	}
	now := time.Now().UTC()
	rs.Time = &now
}

// Run commits deferred system setup and starts the container.
func (seal *Outcome) Run(rs *RunState) error {
	if !seal.f.CompareAndSwap(false, true) {
		// Run does much more than just starting a process; calling it twice, even if the first call fails, will result
		// in inconsistent state that is impossible to clean up; return here to limit damage and hopefully give the
		// other Run a chance to return
		return errors.New("outcome: attempted to run twice")
	}

	if rs == nil {
		panic("invalid state")
	}

	// read comp value early to allow for early failure
	hsuPath := internal.MustHsuPath()

	if err := seal.sys.Commit(seal.ctx); err != nil {
		return err
	}
	store := state.NewMulti(seal.runDirPath.String())
	deferredStoreFunc := func(c state.Cursor) error { return nil } // noop until state in store
	defer func() {
		var revertErr error
		storeErr := new(StateStoreError)
		storeErr.Inner, storeErr.DoErr = store.Do(seal.user.identity.unwrap(), func(c state.Cursor) {
			revertErr = func() error {
				storeErr.InnerErr = deferredStoreFunc(c)

				var rt system.Enablement
				ec := system.Process
				if states, err := c.Load(); err != nil {
					// revert per-process state here to limit damage
					storeErr.OpErr = err
					return seal.sys.Revert((*system.Criteria)(&ec))
				} else {
					if l := len(states); l == 0 {
						ec |= system.User
					} else {
						hlog.Verbosef("found %d instances, cleaning up without user-scoped operations", l)
					}

					// accumulate enablements of remaining launchers
					for i, s := range states {
						if s.Config != nil {
							rt |= s.Config.Enablements.Unwrap()
						} else {
							log.Printf("state entry %d does not contain config", i)
						}
					}
				}
				ec |= rt ^ (system.EWayland | system.EX11 | system.EDBus | system.EPulse)
				if hlog.Load() {
					if ec > 0 {
						hlog.Verbose("reverting operations scope", system.TypeString(ec))
					}
				}

				return seal.sys.Revert((*system.Criteria)(&ec))
			}()
		})
		storeErr.save(revertErr, store.Close())
		rs.RevertErr = storeErr.equiv("error during cleanup:")
	}()

	ctx, cancel := context.WithCancel(seal.ctx)
	defer cancel()
	cmd := exec.CommandContext(ctx, hsuPath)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Dir = container.FHSRoot // container init enters final working directory
	// shim runs in the same session as monitor; see shim.go for behaviour
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGCONT) }

	var e *gob.Encoder
	if fd, encoder, err := container.Setup(&cmd.ExtraFiles); err != nil {
		return hlog.WrapErrSuffix(err,
			"cannot create shim setup pipe:")
	} else {
		e = encoder
		cmd.Env = []string{
			// passed through to shim by hsu
			shimEnv + "=" + strconv.Itoa(fd),
			// interpreted by hsu
			"HAKUREI_APP_ID=" + seal.user.identity.String(),
		}
	}

	if len(seal.user.supp) > 0 {
		hlog.Verbosef("attaching supplementary group ids %s", seal.user.supp)
		// interpreted by hsu
		cmd.Env = append(cmd.Env, "HAKUREI_GROUPS="+strings.Join(seal.user.supp, " "))
	}

	hlog.Verbosef("setuid helper at %s", hsuPath)
	hlog.Suspend()
	if err := cmd.Start(); err != nil {
		return hlog.WrapErrSuffix(err,
			"cannot start setuid wrapper:")
	}
	rs.setStart()

	// this prevents blocking forever on an early failure
	waitErr, setupErr := make(chan error, 1), make(chan error, 1)
	go func() { waitErr <- cmd.Wait(); cancel() }()
	go func() {
		setupErr <- e.Encode(&shimParams{
			os.Getpid(),
			seal.waitDelay,
			seal.container,
			hlog.Load(),
		})
	}()

	select {
	case err := <-setupErr:
		if err != nil {
			hlog.Resume()
			return hlog.WrapErrSuffix(err,
				"cannot transmit shim config:")
		}

	case <-ctx.Done():
		hlog.Resume()
		return hlog.WrapErr(syscall.ECANCELED,
			"shim setup canceled")
	}

	// returned after blocking on waitErr
	var earlyStoreErr = new(StateStoreError)
	{
		// shim accepted setup payload, create process state
		sd := state.State{
			ID:   seal.id.unwrap(),
			PID:  cmd.Process.Pid,
			Time: *rs.Time,
		}
		earlyStoreErr.Inner, earlyStoreErr.DoErr = store.Do(seal.user.identity.unwrap(), func(c state.Cursor) {
			earlyStoreErr.InnerErr = c.Save(&sd, seal.ct)
		})
	}

	// state in store at this point, destroy defunct state entry on return
	deferredStoreFunc = func(c state.Cursor) error { return c.Destroy(seal.id.unwrap()) }

	waitTimeout := make(chan struct{})
	go func() { <-seal.ctx.Done(); time.Sleep(shimWaitTimeout); close(waitTimeout) }()

	select {
	case rs.WaitErr = <-waitErr:
		rs.WaitStatus = cmd.ProcessState.Sys().(syscall.WaitStatus)
		if hlog.Load() {
			switch {
			case rs.Exited():
				hlog.Verbosef("process %d exited with code %d", cmd.Process.Pid, rs.ExitStatus())

			case rs.CoreDump():
				hlog.Verbosef("process %d dumped core", cmd.Process.Pid)

			case rs.Signaled():
				hlog.Verbosef("process %d got %s", cmd.Process.Pid, rs.Signal())

			default:
				hlog.Verbosef("process %d exited with status %#x", cmd.Process.Pid, rs.WaitStatus)
			}
		}
	case <-waitTimeout:
		rs.WaitErr = syscall.ETIMEDOUT
		hlog.Resume()
		log.Printf("process %d did not terminate", cmd.Process.Pid)
	}

	hlog.Resume()
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
