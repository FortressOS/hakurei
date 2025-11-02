package outcome

import (
	"context"
	"encoding/gob"
	"errors"
	"math"
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
	"hakurei.app/internal/store"
	"hakurei.app/message"
	"hakurei.app/system"
)

const (
	// Duration to wait for shim to exit on top of container WaitDelay.
	shimWaitTimeout = 5 * time.Second
	// Timeout for writing outcomeState to the shim setup pipe.
	shimSetupTimeout = 5 * time.Second
)

// NewStore returns the address of a new instance of [store.Store].
func NewStore(sc *hst.Paths) *store.Store { return store.New(sc.SharePath.Append("state")) }

// main carries out outcome and terminates. main does not return.
func (k *outcome) main(msg message.Msg) {
	if k.ctx == nil || k.sys == nil || k.state == nil {
		panic("outcome: did not finalise")
	}

	// read comp value early for early failure
	hsuPath := internal.MustHsuPath()

	const (
		// transitions to processCommit, or processFinal on failure
		processStart = iota
		// transitions to processServe, or processLifecycle on failure
		processCommit
		// transitions to processLifecycle only
		processServe
		// transitions to processCleanup only
		processLifecycle
		// transitions to processFinal only
		processCleanup
		// execution terminates, must be the final state
		processFinal
	)

	// for the shim process
	ctx, cancel := context.WithCancel(k.ctx)
	defer cancel()

	var (
		// state for next iteration
		processState uintptr = processStart
		// current state, must not be mutated directly
		processStateCur uintptr = math.MaxUint
		// point in time the current iteration began
		processTime time.Time

		// whether sys is currently in between a call to Commit and Revert
		isBeforeRevert bool

		// initialised during processStart if successful
		handle *store.Handle
		// initialised during processServe if state is saved
		entryHandle *store.EntryHandle

		// can be set in any state, used in processFinal
		exitCode int

		// shim process startup time,
		// populated in processStart, accessed by processServe
		startTime time.Time
		// shim process as target uid,
		// populated in processStart, accessed by processServe
		shimCmd *exec.Cmd
		// write end of shim setup pipe,
		// populated in processStart, accessed by processServe
		shimPipe *os.File

		// perror cancels ctx and prints an error message
		perror = func(err error, message string) {
			cancel()
			if shimPipe != nil {
				if closeErr := shimPipe.Close(); closeErr != nil {
					msg.Verbose(closeErr.Error())
				}
				shimPipe = nil
			}
			if exitCode == 0 {
				exitCode = 1
			}
			printMessageError(msg.GetLogger().Println, "cannot "+message+":", err)
		}

		// perrorFatal cancels ctx, prints an error message, and sets the next state
		perrorFatal = func(err error, message string, newState uintptr) {
			perror(err, message)
			processState = newState
		}
	)

	for {
		var processStatePrev uintptr
		processStatePrev, processStateCur = processStateCur, processState

		if !processTime.IsZero() && processStatePrev != processLifecycle {
			msg.Verbosef("state %d took %.2f ms", processStatePrev, float64(time.Since(processTime).Nanoseconds())/1e6)
		}
		processTime = time.Now()

		switch processState {
		case processStart:
			if h, err := NewStore(&k.state.sc).Handle(k.state.identity.unwrap()); err != nil {
				perrorFatal(err, "obtain store segment handle", processFinal)
				continue
			} else {
				handle = h
			}

			cmd, f, err := k.start(ctx, msg, hsuPath, &startTime)
			if err != nil {
				perrorFatal(err, "start shim", processFinal)
				continue
			} else {
				shimCmd, shimPipe = cmd, f
			}

			processState = processCommit

		case processCommit:
			if isBeforeRevert {
				perrorFatal(newWithMessage("invalid transition to commit state"), "commit", processLifecycle)
				continue
			}

			unlock, err := handle.Lock()
			if err != nil {
				perrorFatal(err, "acquire lock on store segment", processLifecycle)
				continue
			}
			if entryHandle, err = handle.Save(&hst.State{
				ID:      k.state.id.unwrap(),
				PID:     os.Getpid(),
				ShimPID: shimCmd.Process.Pid,
				Config:  k.config,
				Time:    startTime,
			}); err != nil {
				unlock()
				// transition here to avoid the commit/revert cycle on the doomed instance
				perrorFatal(err, "save instance state", processLifecycle)
				continue
			}

			err = k.sys.Commit()
			unlock()
			if err != nil {
				perrorFatal(err, "commit system setup", processLifecycle)
				continue
			}
			isBeforeRevert = true

			processState = processServe

		case processServe:
			// this state transition to processLifecycle only
			processState = processLifecycle

			// this starts the container, system setup must complete before this point
			if err := serveShim(msg, shimPipe, k.state); err != nil {
				perror(err, "serve shim payload")
				continue
			} else {
				shimPipe = nil // this is already closed by serveShim
			}

		case processLifecycle:
			// this state transition to processCleanup only
			processState = processCleanup

			msg.Suspend()
			select {
			case err := <-func() chan error { w := make(chan error, 1); go func() { w <- shimCmd.Wait(); cancel() }(); return w }():
				wstatus, ok := shimCmd.ProcessState.Sys().(syscall.WaitStatus)
				if ok {
					if v := wstatus.ExitStatus(); v != 0 {
						exitCode = v
					}
				}

				if msg.IsVerbose() {
					if !ok {
						if err != nil {
							msg.Verbosef("wait: %v", err)
						}
					} else {
						switch {
						case wstatus.Exited():
							msg.Verbosef("process %d exited with code %d", shimCmd.Process.Pid, wstatus.ExitStatus())

						case wstatus.CoreDump():
							msg.Verbosef("process %d dumped core", shimCmd.Process.Pid)

						case wstatus.Signaled():
							msg.Verbosef("process %d got %s", shimCmd.Process.Pid, wstatus.Signal())

						default:
							msg.Verbosef("process %d exited with status %#x", shimCmd.Process.Pid, wstatus)
						}
					}
				}

			case <-func() chan struct{} {
				w := make(chan struct{})
				// this ties processLifecycle to ctx with the additional compensated timeout duration
				// to allow transition to the next state on a locked up shim
				go func() { <-ctx.Done(); time.Sleep(k.state.Shim.WaitDelay + shimWaitTimeout); close(w) }()
				return w
			}():
				// this is only reachable when wait did not return within shimWaitTimeout, after its WaitDelay has elapsed.
				// This is different from the container failing to terminate within its timeout period, as that is enforced
				// by the shim. This path is instead reached when there is a lockup in shim preventing it from completing.
				msg.GetLogger().Printf("process %d did not terminate", shimCmd.Process.Pid)
			}
			msg.Resume()

		case processCleanup:
			// this state transition to processFinal only
			processState = processFinal

			unlock := func() { msg.Verbose("skipping unlock as lock was not successfully acquired") }
			if f, err := handle.Lock(); err != nil {
				perror(err, "acquire lock on store segment")
			} else {
				unlock = f
			}

			if entryHandle != nil {
				if err := entryHandle.Destroy(); err != nil {
					perror(err, "destroy state entry")
				}
			}

			if isBeforeRevert {
				ec := system.Process

				if entries, _, err := handle.Entries(); err != nil {
					// it is impossible to continue from this point,
					// per-process state will be reverted to limit damage
					perror(err, "read store segment entries")
				} else {
					// accumulate enablements of remaining instances
					var (
						// alive enablement bits
						rt hst.Enablement
						// alive instance count
						n int
					)
					for eh := range entries {
						var et hst.Enablement
						if et, err = eh.Load(nil); err != nil {
							perror(err, "read state header of instance "+eh.ID.String())
						} else {
							rt |= et
							n++
						}
					}

					if n == 0 {
						ec |= system.User
					} else {
						msg.Verbosef("found %d instances, cleaning up without user-scoped operations", n)
					}
					ec |= rt ^ (hst.EWayland | hst.EX11 | hst.EDBus | hst.EPulse)
					if msg.IsVerbose() {
						if ec > 0 {
							msg.Verbose("reverting operations scope", system.TypeString(ec))
						}
					}
				}

				if err := k.sys.Revert((*system.Criteria)(&ec)); err != nil {
					var joinError interface {
						Unwrap() []error
						error
					}
					if !errors.As(err, &joinError) || joinError == nil {
						perror(err, "revert system setup")
					} else {
						for _, v := range joinError.Unwrap() {
							perror(v, "revert system setup step")
						}
					}
				}
				isBeforeRevert = false
			}
			unlock()

		case processFinal:
			msg.BeforeExit()
			os.Exit(exitCode)

		default: // not reached
			k.fatalf("invalid transition from state %d to %d", processStatePrev, processState)
			panic("unreachable")
		}
	}
}

// start starts the shim via cmd/hsu.
//
// If successful, a [time.Time] value for [hst.State] is stored in the value pointed to by startTime.
// The resulting [exec.Cmd] and write end of the shim setup pipe is returned.
func (k *outcome) start(ctx context.Context, msg message.Msg,
	hsuPath *check.Absolute,
	startTime *time.Time,
) (*exec.Cmd, *os.File, error) {
	cmd := exec.CommandContext(ctx, hsuPath.String())
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Dir = fhs.Root // container init enters final working directory
	// shim runs in the same session as monitor; see shim.go for behaviour
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGCONT) }

	var shimPipe *os.File
	if fd, w, err := container.Setup(&cmd.ExtraFiles); err != nil {
		return cmd, nil, &hst.AppError{Step: "create shim setup pipe", Err: err}
	} else {
		shimPipe = w
		cmd.Env = []string{
			// passed through to shim by hsu
			shimEnv + "=" + strconv.Itoa(fd),
			// interpreted by hsu
			"HAKUREI_IDENTITY=" + k.state.identity.String(),
		}
	}

	if len(k.supp) > 0 {
		msg.Verbosef("attaching supplementary group ids %s", k.supp)
		// interpreted by hsu
		cmd.Env = append(cmd.Env, "HAKUREI_GROUPS="+strings.Join(k.supp, " "))
	}

	msg.Verbosef("setuid helper at %s", hsuPath)
	if err := cmd.Start(); err != nil {
		msg.Resume()
		return cmd, shimPipe, &hst.AppError{Step: "start setuid wrapper", Err: err}
	}

	*startTime = time.Now().UTC()
	return cmd, shimPipe, nil
}

// serveShim serves outcomeState through the shim setup pipe.
func serveShim(msg message.Msg, shimPipe *os.File, state *outcomeState) error {
	if shimPipe == nil {
		return newWithMessage("shim pipe not available")
	}

	if err := shimPipe.SetDeadline(time.Now().Add(shimSetupTimeout)); err != nil {
		msg.Verbose(err.Error())
	}
	if err := gob.NewEncoder(shimPipe).Encode(state); err != nil {
		msg.Resume()
		return &hst.AppError{Step: "transmit shim config", Err: err}
	}
	_ = shimPipe.Close()
	return nil
}

// printMessageError prints the error message according to [message.GetMessage],
// or fallback prepended to err if an error message is not available.
func printMessageError(println func(v ...any), fallback string, err error) {
	m, ok := message.GetMessage(err)
	if !ok {
		println(fallback, err)
		return
	}

	println(m)
}
