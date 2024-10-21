package init0

import (
	"encoding/gob"
	"errors"
	"flag"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strconv"
	"syscall"
	"time"

	"git.ophivana.moe/security/fortify/internal/fmsg"
)

const (
	// time to wait for linger processes after death initial process
	residualProcessTimeout = 5 * time.Second
)

// everything beyond this point runs within pid namespace
// proceed with caution!

func doInit(fd uintptr) {
	fmsg.SetPrefix("init")

	// re-exec
	if len(os.Args) > 0 && os.Args[0] != "fortify" && path.IsAbs(os.Args[0]) {
		if err := syscall.Exec(os.Args[0], []string{"fortify", "init"}, os.Environ()); err != nil {
			fmsg.Println("cannot re-exec self:", err)
			// continue anyway
		}
	}

	var payload Payload
	p := os.NewFile(fd, "config-stream")
	if p == nil {
		fmsg.Fatal("invalid config descriptor")
	}
	if err := gob.NewDecoder(p).Decode(&payload); err != nil {
		fmsg.Fatal("cannot decode init payload:", err)
	} else {
		// sharing stdout with parent
		// USE WITH CAUTION
		fmsg.SetVerbose(payload.Verbose)

		// child does not need to see this
		if err = os.Unsetenv(EnvInit); err != nil {
			fmsg.Println("cannot unset", EnvInit+":", err)
			// not fatal
		} else {
			fmsg.VPrintln("received configuration")
		}
	}

	// close config fd
	if err := p.Close(); err != nil {
		fmsg.Println("cannot close config fd:", err)
		// not fatal
	}

	// die with parent
	if _, _, errno := syscall.RawSyscall(syscall.SYS_PRCTL, syscall.PR_SET_PDEATHSIG, uintptr(syscall.SIGKILL), 0); errno != 0 {
		fmsg.Fatal("prctl(PR_SET_PDEATHSIG, SIGKILL):", errno.Error())
	}

	cmd := exec.Command(payload.Argv0)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Args = payload.Argv
	cmd.Env = os.Environ()

	// pass wayland fd
	if payload.WL != -1 {
		if f := os.NewFile(uintptr(payload.WL), "wayland"); f != nil {
			cmd.Env = append(cmd.Env, "WAYLAND_SOCKET="+strconv.Itoa(3+len(cmd.ExtraFiles)))
			cmd.ExtraFiles = append(cmd.ExtraFiles, f)
		}
	}

	if err := cmd.Start(); err != nil {
		fmsg.Fatalf("cannot start %q: %v", payload.Argv0, err)
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

	timeout := make(chan struct{})

	r := 2
	for {
		select {
		case s := <-sig:
			fmsg.VPrintln("received", s.String())
			os.Exit(0)
		case w := <-info:
			if w.wpid == cmd.Process.Pid {
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
			os.Exit(r)
		case <-timeout:
			fmsg.Println("timeout exceeded waiting for lingering processes")
			os.Exit(r)
		}
	}
}

// Try runs init and stops execution if FORTIFY_INIT is set.
func Try() {
	if os.Getpid() != 1 {
		return
	}

	if args := flag.Args(); len(args) == 1 && args[0] == "init" {
		if s, ok := os.LookupEnv(EnvInit); ok {
			if fd, err := strconv.Atoi(s); err != nil {
				fmsg.Fatalf("cannot parse %q: %v", s, err)
			} else {
				doInit(uintptr(fd))
			}
			panic("unreachable")
		}
	}
}
