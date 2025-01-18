package init0

import (
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"syscall"
	"time"

	"git.gensokyo.uk/security/fortify/internal"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
	"git.gensokyo.uk/security/fortify/internal/proc"
)

const (
	// time to wait for linger processes after death of initial process
	residualProcessTimeout = 5 * time.Second
)

// everything beyond this point runs within pid namespace
// proceed with caution!

func Main() {
	// sharing stdout with shim
	// USE WITH CAUTION
	fmsg.SetPrefix("init")

	// setting this prevents ptrace
	if err := internal.PR_SET_DUMPABLE__SUID_DUMP_DISABLE(); err != nil {
		fmsg.Fatalf("cannot set SUID_DUMP_DISABLE: %s", err)
		panic("unreachable")
	}

	if os.Getpid() != 1 {
		fmsg.Fatal("this process must run as pid 1")
		panic("unreachable")
	}

	// re-exec
	if len(os.Args) > 0 && (os.Args[0] != "fortify" || os.Args[1] != "init" || len(os.Args) != 2) && path.IsAbs(os.Args[0]) {
		if err := syscall.Exec(os.Args[0], []string{"fortify", "init"}, os.Environ()); err != nil {
			fmsg.Println("cannot re-exec self:", err)
			// continue anyway
		}
	}

	// receive setup payload
	var (
		payload    Payload
		closeSetup func() error
	)
	if f, err := proc.Receive(Env, &payload); err != nil {
		if errors.Is(err, proc.ErrInvalid) {
			fmsg.Fatal("invalid config descriptor")
		}
		if errors.Is(err, proc.ErrNotSet) {
			fmsg.Fatal("FORTIFY_INIT not set")
		}

		fmsg.Fatalf("cannot decode init setup payload: %v", err)
		panic("unreachable")
	} else {
		fmsg.SetVerbose(payload.Verbose)
		closeSetup = f

		// child does not need to see this
		if err = os.Unsetenv(Env); err != nil {
			fmsg.Printf("cannot unset %s: %v", Env, err)
			// not fatal
		} else {
			fmsg.VPrintln("received configuration")
		}
	}

	// die with parent
	if err := internal.PR_SET_PDEATHSIG__SIGKILL(); err != nil {
		fmsg.Fatalf("prctl(PR_SET_PDEATHSIG, SIGKILL): %v", err)
	}

	cmd := exec.Command(payload.Argv0)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Args = payload.Argv
	cmd.Env = os.Environ()

	if err := cmd.Start(); err != nil {
		fmsg.Fatalf("cannot start %q: %v", payload.Argv0, err)
	}
	fmsg.Suspend()

	// close setup pipe as setup is now complete
	if err := closeSetup(); err != nil {
		fmsg.Println("cannot close setup pipe:", err)
		// not fatal
	}

	sig := make(chan os.Signal, 2)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	type winfo struct {
		wpid    int
		wstatus syscall.WaitStatus
	}
	info := make(chan winfo, 1)
	done := make(chan struct{})

	go func() {
		var (
			err     error
			wpid    = -2
			wstatus syscall.WaitStatus
		)

		// keep going until no child process is left
		for wpid != -1 {
			if err != nil {
				break
			}

			if wpid != -2 {
				info <- winfo{wpid, wstatus}
			}

			err = syscall.EINTR
			for errors.Is(err, syscall.EINTR) {
				wpid, err = syscall.Wait4(-1, &wstatus, 0, nil)
			}
		}
		if !errors.Is(err, syscall.ECHILD) {
			fmsg.Println("unexpected wait4 response:", err)
		}

		close(done)
	}()

	// closed after residualProcessTimeout has elapsed after initial process death
	timeout := make(chan struct{})

	r := 2
	for {
		select {
		case s := <-sig:
			fmsg.VPrintln("received", s.String())
			fmsg.Resume() // output could still be withheld at this point, so resume is called
			fmsg.Exit(0)
		case w := <-info:
			if w.wpid == cmd.Process.Pid {
				// initial process exited, output is most likely available again
				fmsg.Resume()

				switch {
				case w.wstatus.Exited():
					r = w.wstatus.ExitStatus()
				case w.wstatus.Signaled():
					r = 128 + int(w.wstatus.Signal())
				default:
					r = 255
				}

				go func() {
					time.Sleep(residualProcessTimeout)
					close(timeout)
				}()
			}
		case <-done:
			fmsg.Exit(r)
		case <-timeout:
			fmsg.Println("timeout exceeded waiting for lingering processes")
			fmsg.Exit(r)
		}
	}
}
