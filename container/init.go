package container

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
	. "syscall"
	"time"

	"hakurei.app/container/seccomp"
)

const (
	// time to wait for linger processes after death of initial process
	residualProcessTimeout = 5 * time.Second

	/* intermediate tmpfs mount point

	this path might seem like a weird choice, however there are many good reasons to use it:
	- the contents of this path is never exposed to the container:
	  the tmpfs root established here effectively becomes anonymous after pivot_root
	- it is safe to assume this path exists and is a directory:
	  this program will not work correctly without a proper /proc and neither will most others
	- this path belongs to the container init:
	  the container init is not any more privileged or trusted than the rest of the container
	- this path is only accessible by init and root:
	  the container init sets SUID_DUMP_DISABLE and terminates if that fails;

	it should be noted that none of this should become relevant at any point since the resulting
	intermediate root tmpfs should be effectively anonymous */
	intermediateHostPath = "/proc/self/fd"

	// setup params file descriptor
	setupEnv = "HAKUREI_SETUP"
)

type initParams struct {
	Params

	HostUid, HostGid int
	// extra files count
	Count int
	// verbosity pass through
	Verbose bool
}

func Init(prepare func(prefix string), setVerbose func(verbose bool)) {
	runtime.LockOSThread()
	prepare("init")

	if os.Getpid() != 1 {
		log.Fatal("this process must run as pid 1")
	}

	var (
		params      initParams
		closeSetup  func() error
		setupFile   *os.File
		offsetSetup int
	)
	if f, err := Receive(setupEnv, &params, &setupFile); err != nil {
		if errors.Is(err, ErrInvalid) {
			log.Fatal("invalid setup descriptor")
		}
		if errors.Is(err, ErrNotSet) {
			log.Fatal("HAKUREI_SETUP not set")
		}

		log.Fatalf("cannot decode init setup payload: %v", err)
	} else {
		if params.Ops == nil {
			log.Fatal("invalid setup parameters")
		}
		if params.ParentPerm == 0 {
			params.ParentPerm = 0755
		}

		setVerbose(params.Verbose)
		msg.Verbose("received setup parameters")
		closeSetup = f
		offsetSetup = int(setupFile.Fd() + 1)
	}

	// write uid/gid map here so parent does not need to set dumpable
	if err := SetDumpable(SUID_DUMP_USER); err != nil {
		log.Fatalf("cannot set SUID_DUMP_USER: %s", err)
	}
	if err := os.WriteFile("/proc/self/uid_map",
		append([]byte{}, strconv.Itoa(params.Uid)+" "+strconv.Itoa(params.HostUid)+" 1\n"...),
		0); err != nil {
		log.Fatalf("%v", err)
	}
	if err := os.WriteFile("/proc/self/setgroups",
		[]byte("deny\n"),
		0); err != nil && !os.IsNotExist(err) {
		log.Fatalf("%v", err)
	}
	if err := os.WriteFile("/proc/self/gid_map",
		append([]byte{}, strconv.Itoa(params.Gid)+" "+strconv.Itoa(params.HostGid)+" 1\n"...),
		0); err != nil {
		log.Fatalf("%v", err)
	}
	if err := SetDumpable(SUID_DUMP_DISABLE); err != nil {
		log.Fatalf("cannot set SUID_DUMP_DISABLE: %s", err)
	}

	oldmask := Umask(0)
	if params.Hostname != "" {
		if err := Sethostname([]byte(params.Hostname)); err != nil {
			log.Fatalf("cannot set hostname: %v", err)
		}
	}

	// cache sysctl before pivot_root
	LastCap()

	if err := Mount("", "/", "", MS_SILENT|MS_SLAVE|MS_REC, ""); err != nil {
		log.Fatalf("cannot make / rslave: %v", err)
	}

	for i, op := range *params.Ops {
		if op == nil {
			log.Fatalf("invalid op %d", i)
		}

		if err := op.early(&params.Params); err != nil {
			msg.PrintBaseErr(err,
				fmt.Sprintf("cannot prepare op %d:", i))
			msg.BeforeExit()
			os.Exit(1)
		}
	}

	if err := Mount("rootfs", intermediateHostPath, "tmpfs", MS_NODEV|MS_NOSUID, ""); err != nil {
		log.Fatalf("cannot mount intermediate root: %v", err)
	}
	if err := os.Chdir(intermediateHostPath); err != nil {
		log.Fatalf("cannot enter base path: %v", err)
	}

	if err := os.Mkdir(sysrootDir, 0755); err != nil {
		log.Fatalf("%v", err)
	}
	if err := Mount(sysrootDir, sysrootDir, "", MS_SILENT|MS_MGC_VAL|MS_BIND|MS_REC, ""); err != nil {
		log.Fatalf("cannot bind sysroot: %v", err)
	}

	if err := os.Mkdir(hostDir, 0755); err != nil {
		log.Fatalf("%v", err)
	}
	// pivot_root uncovers intermediateHostPath in hostDir
	if err := PivotRoot(intermediateHostPath, hostDir); err != nil {
		log.Fatalf("cannot pivot into intermediate root: %v", err)
	}
	if err := os.Chdir("/"); err != nil {
		log.Fatalf("%v", err)
	}

	for i, op := range *params.Ops {
		// ops already checked during early setup
		msg.Verbosef("%s %s", op.prefix(), op)
		if err := op.apply(&params.Params); err != nil {
			msg.PrintBaseErr(err,
				fmt.Sprintf("cannot apply op %d:", i))
			msg.BeforeExit()
			os.Exit(1)
		}
	}

	// setup requiring host root complete at this point
	if err := Mount(hostDir, hostDir, "", MS_SILENT|MS_REC|MS_PRIVATE, ""); err != nil {
		log.Fatalf("cannot make host root rprivate: %v", err)
	}
	if err := Unmount(hostDir, MNT_DETACH); err != nil {
		log.Fatalf("cannot unmount host root: %v", err)
	}

	{
		var fd int
		if err := IgnoringEINTR(func() (err error) {
			fd, err = Open("/", O_DIRECTORY|O_RDONLY, 0)
			return
		}); err != nil {
			log.Fatalf("cannot open intermediate root: %v", err)
		}
		if err := os.Chdir(sysrootPath); err != nil {
			log.Fatalf("%v", err)
		}

		if err := PivotRoot(".", "."); err != nil {
			log.Fatalf("cannot pivot into sysroot: %v", err)
		}
		if err := Fchdir(fd); err != nil {
			log.Fatalf("cannot re-enter intermediate root: %v", err)
		}
		if err := Unmount(".", MNT_DETACH); err != nil {
			log.Fatalf("cannot unmount intemediate root: %v", err)
		}
		if err := os.Chdir("/"); err != nil {
			log.Fatalf("%v", err)
		}

		if err := Close(fd); err != nil {
			log.Fatalf("cannot close intermediate root: %v", err)
		}
	}

	if _, _, errno := Syscall(SYS_PRCTL, PR_SET_NO_NEW_PRIVS, 1, 0); errno != 0 {
		log.Fatalf("prctl(PR_SET_NO_NEW_PRIVS): %v", errno)
	}

	if _, _, errno := Syscall(SYS_PRCTL, PR_CAP_AMBIENT, PR_CAP_AMBIENT_CLEAR_ALL, 0); errno != 0 {
		log.Fatalf("cannot clear the ambient capability set: %v", errno)
	}
	for i := uintptr(0); i <= LastCap(); i++ {
		if params.Privileged && i == CAP_SYS_ADMIN {
			continue
		}
		if _, _, errno := Syscall(SYS_PRCTL, PR_CAPBSET_DROP, i, 0); errno != 0 {
			log.Fatalf("cannot drop capability from bonding set: %v", errno)
		}
	}

	var keep [2]uint32
	if params.Privileged {
		keep[capToIndex(CAP_SYS_ADMIN)] |= capToMask(CAP_SYS_ADMIN)

		if _, _, errno := Syscall(SYS_PRCTL, PR_CAP_AMBIENT, PR_CAP_AMBIENT_RAISE, CAP_SYS_ADMIN); errno != 0 {
			log.Fatalf("cannot raise CAP_SYS_ADMIN: %v", errno)
		}
	}
	if err := capset(
		&capHeader{_LINUX_CAPABILITY_VERSION_3, 0},
		&[2]capData{{0, keep[0], keep[0]}, {0, keep[1], keep[1]}},
	); err != nil {
		log.Fatalf("cannot capset: %v", err)
	}

	if !params.SeccompDisable {
		rules := params.SeccompRules
		if len(rules) == 0 { // non-empty rules slice always overrides presets
			msg.Verbosef("resolving presets %#x", params.SeccompPresets)
			rules = seccomp.Preset(params.SeccompPresets, params.SeccompFlags)
		}
		if err := seccomp.Load(rules, params.SeccompFlags); err != nil {
			log.Fatalf("cannot load syscall filter: %v", err)
		}
		msg.Verbosef("%d filter rules loaded", len(rules))
	} else {
		msg.Verbose("syscall filter not configured")
	}

	extraFiles := make([]*os.File, params.Count)
	for i := range extraFiles {
		// setup fd is placed before all extra files
		extraFiles[i] = os.NewFile(uintptr(offsetSetup+i), "extra file "+strconv.Itoa(i))
	}
	Umask(oldmask)

	cmd := exec.Command(params.Path)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Args = params.Args
	cmd.Env = params.Env
	cmd.ExtraFiles = extraFiles
	cmd.Dir = params.Dir

	if err := cmd.Start(); err != nil {
		log.Fatalf("%v", err)
	}
	msg.Suspend()

	if err := closeSetup(); err != nil {
		log.Println("cannot close setup pipe:", err)
		// not fatal
	}

	type winfo struct {
		wpid    int
		wstatus WaitStatus
	}
	info := make(chan winfo, 1)
	done := make(chan struct{})

	go func() {
		var (
			err     error
			wpid    = -2
			wstatus WaitStatus
		)

		// keep going until no child process is left
		for wpid != -1 {
			if err != nil {
				break
			}

			if wpid != -2 {
				info <- winfo{wpid, wstatus}
			}

			err = EINTR
			for errors.Is(err, EINTR) {
				wpid, err = Wait4(-1, &wstatus, 0, nil)
			}
		}
		if !errors.Is(err, ECHILD) {
			log.Println("unexpected wait4 response:", err)
		}

		close(done)
	}()

	// handle signals to dump withheld messages
	sig := make(chan os.Signal, 2)
	signal.Notify(sig, SIGINT, SIGTERM)

	// closed after residualProcessTimeout has elapsed after initial process death
	timeout := make(chan struct{})

	r := 2
	for {
		select {
		case s := <-sig:
			if msg.Resume() {
				msg.Verbosef("terminating on %s after process start", s.String())
			} else {
				msg.Verbosef("terminating on %s", s.String())
			}
			os.Exit(0)
		case w := <-info:
			if w.wpid == cmd.Process.Pid {
				// initial process exited, output is most likely available again
				msg.Resume()

				switch {
				case w.wstatus.Exited():
					r = w.wstatus.ExitStatus()
					msg.Verbosef("initial process exited with code %d", w.wstatus.ExitStatus())
				case w.wstatus.Signaled():
					r = 128 + int(w.wstatus.Signal())
					msg.Verbosef("initial process exited with signal %s", w.wstatus.Signal())
				default:
					r = 255
					msg.Verbosef("initial process exited with status %#x", w.wstatus)
				}

				go func() {
					time.Sleep(residualProcessTimeout)
					close(timeout)
				}()
			}
		case <-done:
			msg.BeforeExit()
			os.Exit(r)
		case <-timeout:
			log.Println("timeout exceeded waiting for lingering processes")
			msg.BeforeExit()
			os.Exit(r)
		}
	}
}

// TryArgv0 calls [Init] if the last element of argv0 is "init".
func TryArgv0(v Msg, prepare func(prefix string), setVerbose func(verbose bool)) {
	if len(os.Args) > 0 && path.Base(os.Args[0]) == "init" {
		msg = v
		Init(prepare, setVerbose)
		msg.BeforeExit()
		os.Exit(0)
	}
}
