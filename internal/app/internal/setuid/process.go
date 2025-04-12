package setuid

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

	"git.gensokyo.uk/security/fortify/internal"
	. "git.gensokyo.uk/security/fortify/internal/app"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
	"git.gensokyo.uk/security/fortify/internal/state"
	"git.gensokyo.uk/security/fortify/sandbox"
	"git.gensokyo.uk/security/fortify/system"
)

const shimWaitTimeout = 5 * time.Second

func (seal *outcome) Run(rs *RunState) error {
	if !seal.f.CompareAndSwap(false, true) {
		// run does much more than just starting a process; calling it twice, even if the first call fails, will result
		// in inconsistent state that is impossible to clean up; return here to limit damage and hopefully give the
		// other Run a chance to return
		return errors.New("outcome: attempted to run twice")
	}

	if rs == nil {
		panic("invalid state")
	}

	// read comp value early to allow for early failure
	fsuPath := internal.MustFsuPath()

	if err := seal.sys.Commit(seal.ctx); err != nil {
		return err
	}
	store := state.NewMulti(seal.runDirPath)
	deferredStoreFunc := func(c state.Cursor) error { return nil } // noop until state in store
	defer func() {
		var revertErr error
		storeErr := new(StateStoreError)
		storeErr.Inner, storeErr.DoErr = store.Do(seal.user.aid.unwrap(), func(c state.Cursor) {
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
						fmsg.Verbosef("found %d instances, cleaning up without user-scoped operations", l)
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
						fmsg.Verbose("reverting operations scope", system.TypeString(ec))
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
	cmd := exec.CommandContext(ctx, fsuPath)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Dir = "/" // container init enters final working directory
	// shim runs in the same session as monitor; see shim.go for behaviour
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGCONT) }

	var e *gob.Encoder
	if fd, encoder, err := sandbox.Setup(&cmd.ExtraFiles); err != nil {
		return fmsg.WrapErrorSuffix(err,
			"cannot create shim setup pipe:")
	} else {
		e = encoder
		cmd.Env = []string{
			// passed through to shim by fsu
			shimEnv + "=" + strconv.Itoa(fd),
			// interpreted by fsu
			"FORTIFY_APP_ID=" + seal.user.aid.String(),
		}
	}

	if len(seal.user.supp) > 0 {
		fmsg.Verbosef("attaching supplementary group ids %s", seal.user.supp)
		// interpreted by fsu
		cmd.Env = append(cmd.Env, "FORTIFY_GROUPS="+strings.Join(seal.user.supp, " "))
	}

	fmsg.Verbosef("setuid helper at %s", fsuPath)
	fmsg.Suspend()
	if err := cmd.Start(); err != nil {
		return fmsg.WrapErrorSuffix(err,
			"cannot start setuid wrapper:")
	}
	rs.SetStart()

	// this prevents blocking forever on an early failure
	waitErr, setupErr := make(chan error, 1), make(chan error, 1)
	go func() { waitErr <- cmd.Wait(); cancel() }()
	go func() { setupErr <- e.Encode(&shimParams{os.Getpid(), seal.container, seal.user.data, fmsg.Load()}) }()

	select {
	case err := <-setupErr:
		if err != nil {
			fmsg.Resume()
			return fmsg.WrapErrorSuffix(err,
				"cannot transmit shim config:")
		}

	case <-ctx.Done():
		fmsg.Resume()
		return fmsg.WrapError(syscall.ECANCELED,
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
		earlyStoreErr.Inner, earlyStoreErr.DoErr = store.Do(seal.user.aid.unwrap(), func(c state.Cursor) {
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
		if fmsg.Load() {
			switch {
			case rs.Exited():
				fmsg.Verbosef("process %d exited with code %d", cmd.Process.Pid, rs.ExitStatus())
			case rs.CoreDump():
				fmsg.Verbosef("process %d dumped core", cmd.Process.Pid)
			case rs.Signaled():
				fmsg.Verbosef("process %d got %s", cmd.Process.Pid, rs.Signal())
			default:
				fmsg.Verbosef("process %d exited with status %#x", cmd.Process.Pid, rs.WaitStatus)
			}
		}
	case <-waitTimeout:
		rs.WaitErr = syscall.ETIMEDOUT
		fmsg.Resume()
		log.Printf("process %d did not terminate", cmd.Process.Pid)
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
