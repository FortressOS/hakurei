package init0

import (
	"errors"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"git.gensokyo.uk/security/fortify/helper/proc"
	"git.gensokyo.uk/security/fortify/internal"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
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
	fmsg.Prepare("init")

	// setting this prevents ptrace
	if err := internal.PR_SET_DUMPABLE__SUID_DUMP_DISABLE(); err != nil {
		log.Fatalf("cannot set SUID_DUMP_DISABLE: %s", err)
	}

	if os.Getpid() != 1 {
		log.Fatal("this process must run as pid 1")
	}

	// receive setup payload
	var (
		payload    Payload
		closeSetup func() error
	)
	if f, err := proc.Receive(Env, &payload); err != nil {
		if errors.Is(err, proc.ErrInvalid) {
			log.Fatal("invalid config descriptor")
		}
		if errors.Is(err, proc.ErrNotSet) {
			log.Fatal("FORTIFY_INIT not set")
		}

		log.Fatalf("cannot decode init setup payload: %v", err)
	} else {
		fmsg.Store(payload.Verbose)
		closeSetup = f

		// child does not need to see this
		if err = os.Unsetenv(Env); err != nil {
			log.Printf("cannot unset %s: %v", Env, err)
			// not fatal
		} else {
			fmsg.Verbose("received configuration")
		}
	}

	// die with parent
	if err := internal.PR_SET_PDEATHSIG__SIGKILL(); err != nil {
		log.Fatalf("prctl(PR_SET_PDEATHSIG, SIGKILL): %v", err)
	}

	cmd := exec.Command(payload.Argv0)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Args = payload.Argv
	cmd.Env = os.Environ()

	if err := cmd.Start(); err != nil {
		log.Fatalf("cannot start %q: %v", payload.Argv0, err)
	}
	fmsg.Suspend()

	// close setup pipe as setup is now complete
	if err := closeSetup(); err != nil {
		log.Println("cannot close setup pipe:", err)
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
			log.Println("unexpected wait4 response:", err)
		}

		close(done)
	}()

	// closed after residualProcessTimeout has elapsed after initial process death
	timeout := make(chan struct{})

	r := 2
	for {
		select {
		case s := <-sig:
			if fmsg.Resume() {
				fmsg.Verbosef("terminating on %s after process start", s.String())
			} else {
				fmsg.Verbosef("terminating on %s", s.String())
			}
			internal.Exit(0)
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
			internal.Exit(r)
		case <-timeout:
			log.Println("timeout exceeded waiting for lingering processes")
			internal.Exit(r)
		}
	}
}
