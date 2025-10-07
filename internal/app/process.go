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
	"hakurei.app/container/check"
	"hakurei.app/container/fhs"
	"hakurei.app/hst"
	"hakurei.app/internal"
	"hakurei.app/internal/app/state"
	"hakurei.app/system"
)

// Duration to wait for shim to exit on top of container WaitDelay.
const shimWaitTimeout = 5 * time.Second

// mainState holds persistent state bound to outcome.main.
type mainState struct {
	// done is whether beforeExit has been called already.
	done bool

	// Time is the exact point in time where the process was created.
	// Location must be set to UTC.
	//
	// Time is nil if no process was ever created.
	Time *time.Time

	store   state.Store
	cancel  context.CancelFunc
	cmd     *exec.Cmd
	cmdWait chan error

	k *outcome
	container.Msg
	uintptr
	*finaliseProcess
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
	defer ms.BeforeExit()

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

		// this ties waitDone to ctx with the additional compensated timeout duration
		go func() { <-ms.k.ctx.Done(); time.Sleep(ms.waitDelay + shimWaitTimeout); close(waitDone) }()

		select {
		case err := <-ms.cmdWait:
			wstatus, ok := ms.cmd.ProcessState.Sys().(syscall.WaitStatus)
			if ok {
				if v := wstatus.ExitStatus(); v != 0 {
					hasErr = true
					exitCode = v
				}
			}

			if ms.IsVerbose() {
				if !ok {
					if err != nil {
						ms.Verbosef("wait: %v", err)
					}
				} else {
					switch {
					case wstatus.Exited():
						ms.Verbosef("process %d exited with code %d", ms.cmd.Process.Pid, wstatus.ExitStatus())

					case wstatus.CoreDump():
						ms.Verbosef("process %d dumped core", ms.cmd.Process.Pid)

					case wstatus.Signaled():
						ms.Verbosef("process %d got %s", ms.cmd.Process.Pid, wstatus.Signal())

					default:
						ms.Verbosef("process %d exited with status %#x", ms.cmd.Process.Pid, wstatus)
					}
				}
			}

		case <-waitDone:
			ms.Resume()
			// this is only reachable when shim did not exit within shimWaitTimeout, after its WaitDelay has elapsed.
			// This is different from the container failing to terminate within its timeout period, as that is enforced
			// by the shim. This path is instead reached when there is a lockup in shim preventing it from completing.
			log.Printf("process %d did not terminate", ms.cmd.Process.Pid)
		}

		ms.Resume()
	}

	if ms.uintptr&mainNeedsRevert != 0 {
		if ok, err := ms.store.Do(ms.identity.unwrap(), func(c state.Cursor) {
			if ms.uintptr&mainNeedsDestroy != 0 {
				if err := c.Destroy(ms.id.unwrap()); err != nil {
					perror(err, "destroy state entry")
				}
			}

			var rt hst.Enablement
			if states, err := c.Load(); err != nil {
				// it is impossible to continue from this point;
				// revert per-process state here to limit damage
				ec := system.Process
				if revertErr := ms.k.sys.Revert((*system.Criteria)(&ec)); revertErr != nil {
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
					ms.Verbosef("found %d instances, cleaning up without user-scoped operations", l)
				}

				// accumulate enablements of remaining launchers
				for i, s := range states {
					if s.Config != nil {
						rt |= s.Config.Enablements.Unwrap()
					} else {
						log.Printf("state entry %d does not contain config", i)
					}
				}

				ec |= rt ^ (hst.EWayland | hst.EX11 | hst.EDBus | hst.EPulse)
				if ms.IsVerbose() {
					if ec > 0 {
						ms.Verbose("reverting operations scope", system.TypeString(ec))
					}
				}

				if err = ms.k.sys.Revert((*system.Criteria)(&ec)); err != nil {
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

// finaliseProcess contains information collected during outcome.finalise used in outcome.main.
type finaliseProcess struct {
	// Supplementary group ids.
	supp []string

	// Copied from [hst.ContainerConfig], without exceeding [MaxShimWaitDelay].
	waitDelay time.Duration

	// Copied from the RunDirPath field of [hst.Paths].
	runDirPath *check.Absolute

	// Copied from outcomeState.
	identity *stringPair[int]

	// Copied from outcomeState.
	id *stringPair[state.ID]
}

// main carries out outcome and terminates. main does not return.
func (k *outcome) main(msg container.Msg) {
	if !k.active.CompareAndSwap(false, true) {
		panic("outcome: attempted to run twice")
	}

	if k.proc == nil {
		panic("outcome: did not finalise")
	}

	// read comp value early for early failure
	hsuPath := internal.MustHsuPath()

	// ms.beforeExit required beyond this point
	ms := &mainState{Msg: msg, k: k, finaliseProcess: k.proc}

	if err := k.sys.Commit(); err != nil {
		ms.fatal("cannot commit system setup:", err)
	}
	ms.uintptr |= mainNeedsRevert
	ms.store = state.NewMulti(msg, ms.runDirPath.String())

	ctx, cancel := context.WithCancel(k.ctx)
	defer cancel()
	ms.cancel = cancel

	ms.cmd = exec.CommandContext(ctx, hsuPath.String())
	ms.cmd.Stdin, ms.cmd.Stdout, ms.cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	ms.cmd.Dir = fhs.Root // container init enters final working directory
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
			"HAKUREI_IDENTITY=" + ms.identity.String(),
		}
	}

	if len(ms.supp) > 0 {
		msg.Verbosef("attaching supplementary group ids %s", ms.supp)
		// interpreted by hsu
		ms.cmd.Env = append(ms.cmd.Env, "HAKUREI_GROUPS="+strings.Join(ms.supp, " "))
	}

	msg.Verbosef("setuid helper at %s", hsuPath)
	msg.Suspend()
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
				ms.waitDelay,
				&k.container,
				msg.IsVerbose(),
			})
		}()
		return
	}():
		if err != nil {
			msg.Resume()
			ms.fatal("cannot transmit shim config:", err)
		}

	case <-ctx.Done():
		msg.Resume()
		ms.fatal("shim context canceled:", newWithMessageError("shim setup canceled", ctx.Err()))
	}

	// shim accepted setup payload, create process state
	if ok, err := ms.store.Do(ms.identity.unwrap(), func(c state.Cursor) {
		if err := c.Save(&state.State{
			ID:   ms.id.unwrap(),
			PID:  ms.cmd.Process.Pid,
			Time: *ms.Time,
		}, k.ct); err != nil {
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
