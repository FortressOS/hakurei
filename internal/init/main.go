package init0

import (
	"encoding/gob"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strconv"
	"syscall"
	"time"

	"git.ophivana.moe/security/fortify/internal/verbose"
)

const (
	// time to wait for linger processes after death initial process
	residualProcessTimeout = 5 * time.Second
)

// everything beyond this point runs within pid namespace
// proceed with caution!

func doInit(fd uintptr) {
	// re-exec
	if len(os.Args) > 0 && os.Args[0] != "fortify" && path.IsAbs(os.Args[0]) {
		if err := syscall.Exec(os.Args[0], []string{"fortify", "init"}, os.Environ()); err != nil {
			fmt.Println("fortify-init: cannot re-exec self:", err)
			// continue anyway
		}
	}

	verbose.Prefix = "fortify-init:"

	var payload Payload
	p := os.NewFile(fd, "config-stream")
	if p == nil {
		fmt.Println("fortify-init: invalid config descriptor")
		os.Exit(1)
	}
	if err := gob.NewDecoder(p).Decode(&payload); err != nil {
		fmt.Println("fortify-init: cannot decode init payload:", err)
		os.Exit(1)
	} else {
		// sharing stdout with parent
		// USE WITH CAUTION
		verbose.Set(payload.Verbose)

		// child does not need to see this
		if err = os.Unsetenv(EnvInit); err != nil {
			fmt.Println("fortify-init: cannot unset", EnvInit+":", err)
			// not fatal
		} else {
			verbose.Println("received configuration")
		}
	}

	// close config fd
	if err := p.Close(); err != nil {
		fmt.Println("fortify-init: cannot close config fd:", err)
		// not fatal
	}

	// die with parent
	if _, _, errno := syscall.RawSyscall(syscall.SYS_PRCTL, syscall.PR_SET_PDEATHSIG, uintptr(syscall.SIGKILL), 0); errno != 0 {
		fmt.Println("fortify-init: prctl(PR_SET_PDEATHSIG, SIGKILL):", errno.Error())
		os.Exit(1)
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
		fmt.Printf("fortify-init: cannot start %q: %v", payload.Argv0, err)
		os.Exit(1)
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
			fmt.Println("fortify-init: unexpected wait4 response:", err)
		}

		close(done)
	}()

	timeout := make(chan struct{})

	r := 2
	for {
		select {
		case s := <-sig:
			verbose.Println("received", s.String())
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
			fmt.Println("fortify-init: timeout exceeded waiting for lingering processes")
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
				fmt.Printf("fortify-init: cannot parse %q: %v", s, err)
				os.Exit(1)
			} else {
				doInit(uintptr(fd))
			}
			panic("unreachable")
		}
	}
}
