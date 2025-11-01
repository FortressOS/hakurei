package outcome

import (
	"context"
	"encoding/gob"
	"errors"
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

// mainState holds persistent state bound to outcome.main.
type mainState struct {
	// done is whether beforeExit has been called already.
	done bool

	// Populated on successful hsu startup.
	cmd *exec.Cmd
	// Cancels cmd, must be populated before cmd is populated.
	cancel context.CancelFunc

	store store.Compat

	k *outcome
	message.Msg
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
	defer ms.BeforeExit()

	if isFault && ms.cancel != nil {
		ms.cancel()
	}

	var hasErr bool
	// updates hasErr but does not terminate
	perror := func(err error, message string) {
		hasErr = true
		printMessageError(ms.GetLogger().Println, "cannot "+message+":", err)
	}
	exitCode := 1
	defer func() {
		if hasErr {
			os.Exit(exitCode)
		}
	}()

	// this also handles wait for a non-fault termination
	if ms.cmd != nil {
		select {
		case err := <-func() chan error { w := make(chan error, 1); go func() { w <- ms.cmd.Wait(); ms.cancel() }(); return w }():
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

		case <-func() chan struct{} {
			w := make(chan struct{})
			// this ties waitDone to ctx with the additional compensated timeout duration
			go func() { <-ms.k.ctx.Done(); time.Sleep(ms.k.state.Shim.WaitDelay + shimWaitTimeout); close(w) }()
			return w
		}():
			ms.Resume()
			// this is only reachable when shim did not exit within shimWaitTimeout, after its WaitDelay has elapsed.
			// This is different from the container failing to terminate within its timeout period, as that is enforced
			// by the shim. This path is instead reached when there is a lockup in shim preventing it from completing.
			ms.GetLogger().Printf("process %d did not terminate", ms.cmd.Process.Pid)
		}

		ms.Resume()
	}

	if ms.uintptr&mainNeedsRevert != 0 {
		if ok, err := ms.store.Do(ms.k.state.identity.unwrap(), func(c store.Cursor) {
			if ms.uintptr&mainNeedsDestroy != 0 {
				if err := c.Destroy(ms.k.state.id.unwrap()); err != nil {
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
						ms.GetLogger().Printf("state entry %d does not contain config", i)
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
}

// fatal calls printMessageError, performs necessary cleanup, followed by a call to [os.Exit](1).
func (ms mainState) fatal(fallback string, ferr error) {
	printMessageError(ms.GetLogger().Println, fallback, ferr)
	ms.beforeExit(true)
	os.Exit(1)
}

// main carries out outcome and terminates. main does not return.
func (k *outcome) main(msg message.Msg) {
	if k.ctx == nil || k.sys == nil || k.state == nil {
		panic("outcome: did not finalise")
	}

	// read comp value early for early failure
	hsuPath := internal.MustHsuPath()

	// ms.beforeExit required beyond this point
	ms := mainState{Msg: msg, k: k}

	if err := k.sys.Commit(); err != nil {
		ms.fatal("cannot commit system setup:", err)
	}
	ms.uintptr |= mainNeedsRevert
	ms.store = store.NewMulti(msg, k.state.sc.RunDirPath)

	ctx, cancel := context.WithCancel(k.ctx)
	defer cancel()
	ms.cancel = cancel

	// shim starts and blocks on setup payload before container is started
	var (
		startTime time.Time
		shimPipe  *os.File
	)
	if cmd, f, err := k.start(ctx, msg, hsuPath, &startTime); err != nil {
		ms.fatal("cannot start shim:", err)
		panic("unreachable")
	} else {
		ms.cmd, shimPipe = cmd, f
	}

	// this starts the container, system setup must complete before this point
	if err := serveShim(msg, shimPipe, k.state); err != nil {
		ms.fatal("cannot serve shim payload:", err)
	}

	// shim accepted setup payload, create process state
	if ok, err := ms.store.Do(k.state.identity.unwrap(), func(c store.Cursor) {
		if err := c.Save(&hst.State{
			ID:      k.state.id.unwrap(),
			PID:     os.Getpid(),
			ShimPID: ms.cmd.Process.Pid,
			Config:  k.config,
			Time:    startTime,
		}); err != nil {
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
	msg.Suspend()
	if err := cmd.Start(); err != nil {
		msg.Resume()
		return cmd, shimPipe, &hst.AppError{Step: "start setuid wrapper", Err: err}
	}

	*startTime = time.Now().UTC()
	return cmd, shimPipe, nil
}

// serveShim serves outcomeState through the shim setup pipe.
func serveShim(msg message.Msg, shimPipe *os.File, state *outcomeState) error {
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
