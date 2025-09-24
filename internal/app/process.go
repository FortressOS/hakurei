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

// duration to wait for shim to exit, after container WaitDelay has elapsed.
const shimWaitTimeout = 5 * time.Second

// mainState holds persistent state bound to [Outcome.Main].
type mainState struct {
	// done is whether beforeExit has been called already.
	done bool

	// Time is the exact point in time where the process was created.
	// Location must be set to UTC.
	//
	// Time is nil if no process was ever created.
	Time *time.Time

	seal    *Outcome
	store   state.Store
	cancel  context.CancelFunc
	cmd     *exec.Cmd
	cmdWait chan error

	uintptr
}

const (
	// mainNeedsRevert indicates the call to Commit has succeeded.
	mainNeedsRevert uintptr = 1 << iota
	// mainNeedsDestroy indicates the instance state entry is present in the store.
	mainNeedsDestroy
)

// beforeExit must be called immediately before a call to [os.Exit].
func (ms mainState) beforeExit(isFault bool) {
	if ms.done {
		panic("attempting to call beforeExit twice")
	}
	ms.done = true
	defer hlog.BeforeExit()

	if isFault && ms.cancel != nil {
		ms.cancel()
	}

	var hasErr bool
	// updates hasErr but does not terminate
	perror := func(err error, message string) {
		hasErr = true
		printMessageError("cannot "+message+":", err)
	}
	exitCode := 1
	defer func() {
		if hasErr {
			os.Exit(exitCode)
		}
	}()

	// this also handles wait for a non-fault termination
	if ms.cmd != nil && ms.cmdWait != nil {
		waitDone := make(chan struct{})
		// TODO(ophestra): enforce this limit early so it does not have to be done twice
		shimTimeoutCompensated := shimWaitTimeout
		if ms.seal.waitDelay > MaxShimWaitDelay {
			shimTimeoutCompensated += MaxShimWaitDelay
		} else {
			shimTimeoutCompensated += ms.seal.waitDelay
		}
		// this ties waitDone to ctx with the additional compensated timeout duration
		go func() { <-ms.seal.ctx.Done(); time.Sleep(shimTimeoutCompensated); close(waitDone) }()

		select {
		case err := <-ms.cmdWait:
			wstatus, ok := ms.cmd.ProcessState.Sys().(syscall.WaitStatus)
			if ok {
				if v := wstatus.ExitStatus(); v != 0 {
					hasErr = true
					exitCode = v
				}
			}

			if hlog.Load() {
				if !ok {
					if err != nil {
						hlog.Verbosef("wait: %v", err)
					}
				} else {
					switch {
					case wstatus.Exited():
						hlog.Verbosef("process %d exited with code %d", ms.cmd.Process.Pid, wstatus.ExitStatus())

					case wstatus.CoreDump():
						hlog.Verbosef("process %d dumped core", ms.cmd.Process.Pid)

					case wstatus.Signaled():
						hlog.Verbosef("process %d got %s", ms.cmd.Process.Pid, wstatus.Signal())

					default:
						hlog.Verbosef("process %d exited with status %#x", ms.cmd.Process.Pid, wstatus)
					}
				}
			}

		case <-waitDone:
			hlog.Resume()
			// this is only reachable when shim did not exit within shimWaitTimeout, after its WaitDelay has elapsed.
			// This is different from the container failing to terminate within its timeout period, as that is enforced
			// by the shim. This path is instead reached when there is a lockup in shim preventing it from completing.
			log.Printf("process %d did not terminate", ms.cmd.Process.Pid)
		}

		hlog.Resume()
		if ms.seal.sync != nil {
			if err := ms.seal.sync.Close(); err != nil {
				perror(err, "close wayland security context")
			}
		}
		if ms.seal.dbusMsg != nil {
			ms.seal.dbusMsg()
		}
	}

	if ms.uintptr&mainNeedsRevert != 0 {
		if ok, err := ms.store.Do(ms.seal.user.identity.unwrap(), func(c state.Cursor) {
			if ms.uintptr&mainNeedsDestroy != 0 {
				if err := c.Destroy(ms.seal.id.unwrap()); err != nil {
					perror(err, "destroy state entry")
				}
			}

			var rt system.Enablement
			if states, err := c.Load(); err != nil {
				// it is impossible to continue from this point;
				// revert per-process state here to limit damage
				ec := system.Process
				if revertErr := ms.seal.sys.Revert((*system.Criteria)(&ec)); revertErr != nil {
					var joinError interface {
						Unwrap() []error
						error
					}
					if !errors.As(revertErr, &joinError) || joinError == nil {
						perror(revertErr, "revert system setup")
					} else {
						for _, v := range joinError.Unwrap() {
							perror(v, "revert system setup step")
						}
					}
				}
				perror(err, "load instance states")
			} else {
				ec := system.Process
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

				ec |= rt ^ (system.EWayland | system.EX11 | system.EDBus | system.EPulse)
				if hlog.Load() {
					if ec > 0 {
						hlog.Verbose("reverting operations scope", system.TypeString(ec))
					}
				}

				if err = ms.seal.sys.Revert((*system.Criteria)(&ec)); err != nil {
					perror(err, "revert system setup")
				}
			}
		}); err != nil {
			if ok {
				perror(err, "unlock state store")
			} else {
				perror(err, "open state store")
			}
		}
	} else if ms.uintptr&mainNeedsDestroy != 0 {
		panic("unreachable")
	}

	if ms.store != nil {
		if err := ms.store.Close(); err != nil {
			perror(err, "close state store")
		}
	}
}

// fatal calls printMessageError, performs necessary cleanup, followed by a call to [os.Exit](1).
func (ms mainState) fatal(fallback string, ferr error) {
	printMessageError(fallback, ferr)
	ms.beforeExit(true)
	os.Exit(1)
}

// Main commits deferred system setup, runs the container, reverts changes to the system, and terminates the program.
// Main does not return.
func (seal *Outcome) Main() {
	if !seal.f.CompareAndSwap(false, true) {
		panic("outcome: attempted to run twice")
	}

	// read comp value early for early failure
	hsuPath := internal.MustHsuPath()

	// ms.beforeExit required beyond this point
	ms := &mainState{seal: seal}

	if err := seal.sys.Commit(); err != nil {
		ms.fatal("cannot commit system setup:", err)
	}
	ms.uintptr |= mainNeedsRevert
	ms.store = state.NewMulti(seal.runDirPath.String())

	ctx, cancel := context.WithCancel(seal.ctx)
	defer cancel()
	ms.cancel = cancel

	ms.cmd = exec.CommandContext(ctx, hsuPath)
	ms.cmd.Stdin, ms.cmd.Stdout, ms.cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	ms.cmd.Dir = container.FHSRoot // container init enters final working directory
	// shim runs in the same session as monitor; see shim.go for behaviour
	ms.cmd.Cancel = func() error { return ms.cmd.Process.Signal(syscall.SIGCONT) }

	var e *gob.Encoder
	if fd, encoder, err := container.Setup(&ms.cmd.ExtraFiles); err != nil {
		ms.fatal("cannot create shim setup pipe:", err)
	} else {
		e = encoder
		ms.cmd.Env = []string{
			// passed through to shim by hsu
			shimEnv + "=" + strconv.Itoa(fd),
			// interpreted by hsu
			"HAKUREI_APP_ID=" + seal.user.identity.String(),
		}
	}

	if len(seal.user.supp) > 0 {
		hlog.Verbosef("attaching supplementary group ids %s", seal.user.supp)
		// interpreted by hsu
		ms.cmd.Env = append(ms.cmd.Env, "HAKUREI_GROUPS="+strings.Join(seal.user.supp, " "))
	}

	hlog.Verbosef("setuid helper at %s", hsuPath)
	hlog.Suspend()
	if err := ms.cmd.Start(); err != nil {
		ms.fatal("cannot start setuid wrapper:", err)
	}
	startTime := time.Now().UTC()
	ms.cmdWait = make(chan error, 1)
	// this ties context back to the life of the process
	go func() { ms.cmdWait <- ms.cmd.Wait(); cancel() }()
	ms.Time = &startTime

	// unfortunately the I/O here cannot be directly canceled;
	// the cancellation path leads to fatal in this case so that is fine
	select {
	case err := <-func() (setupErr chan error) {
		setupErr = make(chan error, 1)
		go func() {
			setupErr <- e.Encode(&shimParams{
				os.Getpid(),
				seal.waitDelay,
				seal.container,
				hlog.Load(),
			})
		}()
		return
	}():
		if err != nil {
			hlog.Resume()
			ms.fatal("cannot transmit shim config:", err)
		}

	case <-ctx.Done():
		hlog.Resume()
		ms.fatal("shim context canceled:", newWithMessageError("shim setup canceled", ctx.Err()))
	}

	// shim accepted setup payload, create process state
	if ok, err := ms.store.Do(seal.user.identity.unwrap(), func(c state.Cursor) {
		if err := c.Save(&state.State{
			ID:   seal.id.unwrap(),
			PID:  ms.cmd.Process.Pid,
			Time: *ms.Time,
		}, seal.ct); err != nil {
			ms.fatal("cannot save state entry:", err)
		}
	}); err != nil {
		if ok {
			ms.uintptr |= mainNeedsDestroy
			ms.fatal("cannot unlock state store:", err)
		} else {
			ms.fatal("cannot open state store:", err)
		}
	}
	// state in store at this point, destroy defunct state entry on termination
	ms.uintptr |= mainNeedsDestroy

	// beforeExit ties shim process to context
	ms.beforeExit(false)
	os.Exit(0)
}

// printMessageError prints the error message according to [container.GetErrorMessage],
// or fallback prepended to err if an error message is not available.
func printMessageError(fallback string, err error) {
	m, ok := container.GetErrorMessage(err)
	if !ok {
		log.Println(fallback, err)
		return
	}

	log.Print(m)
}
