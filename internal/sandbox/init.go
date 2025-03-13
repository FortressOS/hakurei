package sandbox

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"git.gensokyo.uk/security/fortify/helper/proc"
	"git.gensokyo.uk/security/fortify/helper/seccomp"
	"git.gensokyo.uk/security/fortify/internal"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
)

const (
	// time to wait for linger processes after death of initial process
	residualProcessTimeout = 5 * time.Second

	// intermediate tmpfs mount point
	basePath = "/tmp"

	// setup params file descriptor
	setupEnv = "FORTIFY_SETUP"
)

type initParams struct {
	InitParams

	// extra files count
	Count int
	// verbosity pass through
	Verbose bool
}

func Init(exit func(code int)) {
	runtime.LockOSThread()
	fmsg.Prepare("init")

	if err := internal.SetDumpable(internal.SUID_DUMP_DISABLE); err != nil {
		log.Fatalf("cannot set SUID_DUMP_DISABLE: %s", err)
	}

	if os.Getpid() != 1 {
		log.Fatal("this process must run as pid 1")
	}

	/*
		receive setup payload
	*/

	var (
		params      initParams
		closeSetup  func() error
		setupFile   *os.File
		offsetSetup int
	)
	if f, err := proc.Receive(setupEnv, &params, &setupFile); err != nil {
		if errors.Is(err, proc.ErrInvalid) {
			log.Fatal("invalid setup descriptor")
		}
		if errors.Is(err, proc.ErrNotSet) {
			log.Fatal("FORTIFY_SETUP not set")
		}

		log.Fatalf("cannot decode init setup payload: %v", err)
	} else {
		fmsg.Store(params.Verbose)
		fmsg.Verbose("received setup parameters")
		if params.Verbose {
			seccomp.CPrintln = fmsg.Verbose
		}
		closeSetup = f
		offsetSetup = int(setupFile.Fd() + 1)
	}

	if params.Hostname != "" {
		if err := syscall.Sethostname([]byte(params.Hostname)); err != nil {
			log.Fatalf("cannot set hostname: %v", err)
		}
	}

	/*
		set up mount points from intermediate root
	*/

	if err := syscall.Mount("", "/", "",
		syscall.MS_SILENT|syscall.MS_SLAVE|syscall.MS_REC,
		""); err != nil {
		log.Fatalf("cannot make / rslave: %v", err)
	}

	if err := syscall.Mount("rootfs", basePath, "tmpfs",
		syscall.MS_NODEV|syscall.MS_NOSUID,
		""); err != nil {
		log.Fatalf("cannot mount intermediate root: %v", err)
	}
	if err := os.Chdir(basePath); err != nil {
		log.Fatalf("cannot enter base path: %v", err)
	}

	if err := os.Mkdir(sysrootDir, 0755); err != nil {
		log.Fatalf("%v", err)
	}
	if err := syscall.Mount(sysrootDir, sysrootDir, "",
		syscall.MS_SILENT|syscall.MS_MGC_VAL|syscall.MS_BIND|syscall.MS_REC,
		""); err != nil {
		log.Fatalf("cannot bind sysroot: %v", err)
	}

	if err := os.Mkdir(hostDir, 0755); err != nil {
		log.Fatalf("%v", err)
	}
	if err := syscall.PivotRoot(basePath, hostDir); err != nil {
		log.Fatalf("cannot pivot into intermediate root: %v", err)
	}
	if err := os.Chdir("/"); err != nil {
		log.Fatalf("%v", err)
	}

	for i, op := range *params.Ops {
		fmsg.Verbosef("mounting %s", op)
		if err := op.apply(&params.InitParams); err != nil {
			fmsg.PrintBaseError(err,
				fmt.Sprintf("cannot apply op %d:", i))
			exit(1)
		}
	}

	/*
		pivot to sysroot
	*/

	if err := syscall.Mount(hostDir, hostDir, "",
		syscall.MS_SILENT|syscall.MS_REC|syscall.MS_PRIVATE,
		""); err != nil {
		log.Fatalf("cannot make host root rprivate: %v", err)
	}
	if err := syscall.Unmount(hostDir, syscall.MNT_DETACH); err != nil {
		log.Fatalf("cannot unmount host root: %v", err)
	}

	{
		var fd int
		if err := internal.IgnoringEINTR(func() (err error) {
			fd, err = syscall.Open("/", syscall.O_DIRECTORY|syscall.O_RDONLY, 0)
			return
		}); err != nil {
			log.Fatalf("cannot open intermediate root: %v", err)
		}
		if err := os.Chdir(sysrootPath); err != nil {
			log.Fatalf("%v", err)
		}

		if err := syscall.PivotRoot(".", "."); err != nil {
			log.Fatalf("cannot pivot into sysroot: %v", err)
		}
		if err := syscall.Fchdir(fd); err != nil {
			log.Fatalf("cannot re-enter intermediate root: %v", err)
		}
		if err := syscall.Unmount(".", syscall.MNT_DETACH); err != nil {
			log.Fatalf("cannot unmount intemediate root: %v", err)
		}
		if err := os.Chdir("/"); err != nil {
			log.Fatalf("%v", err)
		}

		if err := syscall.Close(fd); err != nil {
			log.Fatalf("cannot close intermediate root: %v", err)
		}
	}

	/*
		load seccomp filter
	*/

	if _, _, err := syscall.Syscall(PR_SET_NO_NEW_PRIVS, 1, 0, 0); err != 0 {
		log.Fatalf("prctl(PR_SET_NO_NEW_PRIVS): %v", err)
	}
	if err := seccomp.Load(params.Flags.seccomp(params.Seccomp)); err != nil {
		log.Fatalf("cannot load syscall filter: %v", err)
	}

	/* at this point CAP_SYS_ADMIN can be dropped, however it is kept for now as it does not increase attack surface */

	/*
		pass through extra files
	*/

	extraFiles := make([]*os.File, params.Count)
	for i := range extraFiles {
		extraFiles[i] = os.NewFile(uintptr(offsetSetup+i), "extra file "+strconv.Itoa(i))
	}

	/*
		prepare initial process
	*/

	cmd := exec.Command(params.Path)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Args = params.Args
	cmd.Env = params.Env
	cmd.ExtraFiles = extraFiles
	cmd.Dir = params.Dir

	if err := cmd.Start(); err != nil {
		log.Fatalf("%v", err)
	}
	fmsg.Suspend()

	/*
		close setup pipe
	*/

	if err := closeSetup(); err != nil {
		log.Println("cannot close setup pipe:", err)
		// not fatal
	}

	/*
		perform init duties
	*/

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
			exit(0)
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
			exit(r)
		case <-timeout:
			log.Println("timeout exceeded waiting for lingering processes")
			exit(r)
		}
	}
}

// TryArgv0 calls [Init] if the last element of argv0 is "init".
func TryArgv0() {
	if len(os.Args) > 0 && path.Base(os.Args[0]) == "init" {
		Init(internal.Exit)
		internal.Exit(0)
	}
}
