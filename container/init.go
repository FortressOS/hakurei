package container

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"
	"slices"
	"strconv"
	. "syscall"
	"time"

	"hakurei.app/container/seccomp"
)

const (
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
	intermediateHostPath = FHSProc + "self/fd"

	// setup params file descriptor
	setupEnv = "HAKUREI_SETUP"
)

type (
	// Ops is a collection of [Op].
	Ops []Op

	// Op is a generic setup step ran inside the container init.
	// Implementations of this interface are sent as a stream of gobs.
	Op interface {
		// early is called in host root.
		early(state *setupState, k syscallDispatcher) error
		// apply is called in intermediate root.
		apply(state *setupState, k syscallDispatcher) error

		prefix() string
		Is(op Op) bool
		Valid() bool
		fmt.Stringer
	}

	// setupState persists context between Ops.
	setupState struct {
		nonrepeatable uintptr
		*Params
	}
)

// Grow grows the slice Ops points to using [slices.Grow].
func (f *Ops) Grow(n int) { *f = slices.Grow(*f, n) }

const (
	nrAutoEtc = 1 << iota
	nrAutoRoot
)

// initParams are params passed from parent.
type initParams struct {
	Params

	HostUid, HostGid int
	// extra files count
	Count int
	// verbosity pass through
	Verbose bool
}

func Init(prepareLogger func(prefix string), setVerbose func(verbose bool)) {
	initEntrypoint(prepareLogger, setVerbose, direct{})
}

func initEntrypoint(prepareLogger func(prefix string), setVerbose func(verbose bool), k syscallDispatcher) {
	runtime.LockOSThread()
	prepareLogger("init")

	if k.getpid() != 1 {
		k.fatal("this process must run as pid 1")
	}

	if err := k.setPtracer(0); err != nil {
		k.verbosef("cannot enable ptrace protection via Yama LSM: %v", err)
		// not fatal: this program has no additional privileges at initial program start
	}

	var (
		params      initParams
		closeSetup  func() error
		setupFile   *os.File
		offsetSetup int
	)
	if f, err := k.receive(setupEnv, &params, &setupFile); err != nil {
		if errors.Is(err, EBADF) {
			k.fatal("invalid setup descriptor")
		}
		if errors.Is(err, ErrNotSet) {
			k.fatal("HAKUREI_SETUP not set")
		}

		k.fatalf("cannot decode init setup payload: %v", err)
	} else {
		if params.Ops == nil {
			k.fatal("invalid setup parameters")
		}
		if params.ParentPerm == 0 {
			params.ParentPerm = 0755
		}

		setVerbose(params.Verbose)
		k.verbose("received setup parameters")
		closeSetup = f
		offsetSetup = int(setupFile.Fd() + 1)
	}

	// write uid/gid map here so parent does not need to set dumpable
	if err := k.setDumpable(SUID_DUMP_USER); err != nil {
		k.fatalf("cannot set SUID_DUMP_USER: %s", err)
	}
	if err := k.writeFile(FHSProc+"self/uid_map",
		append([]byte{}, strconv.Itoa(params.Uid)+" "+strconv.Itoa(params.HostUid)+" 1\n"...),
		0); err != nil {
		k.fatalf("%v", err)
	}
	if err := k.writeFile(FHSProc+"self/setgroups",
		[]byte("deny\n"),
		0); err != nil && !os.IsNotExist(err) {
		k.fatalf("%v", err)
	}
	if err := k.writeFile(FHSProc+"self/gid_map",
		append([]byte{}, strconv.Itoa(params.Gid)+" "+strconv.Itoa(params.HostGid)+" 1\n"...),
		0); err != nil {
		k.fatalf("%v", err)
	}
	if err := k.setDumpable(SUID_DUMP_DISABLE); err != nil {
		k.fatalf("cannot set SUID_DUMP_DISABLE: %s", err)
	}

	oldmask := k.umask(0)
	if params.Hostname != "" {
		if err := k.sethostname([]byte(params.Hostname)); err != nil {
			k.fatalf("cannot set hostname: %v", err)
		}
	}

	// cache sysctl before pivot_root
	k.lastcap()

	if err := k.mount(zeroString, FHSRoot, zeroString, MS_SILENT|MS_SLAVE|MS_REC, zeroString); err != nil {
		k.fatalf("cannot make / rslave: %v", err)
	}

	state := &setupState{Params: &params.Params}

	/* early is called right before pivot_root into intermediate root;
	this step is mostly for gathering information that would otherwise be difficult to obtain
	via library functions after pivot_root, and implementations are expected to avoid changing
	the state of the mount namespace */
	for i, op := range *params.Ops {
		if op == nil || !op.Valid() {
			k.fatalf("invalid op at index %d", i)
		}

		if err := op.early(state, k); err != nil {
			k.printBaseErr(err,
				fmt.Sprintf("cannot prepare op at index %d:", i))
			k.beforeExit()
			k.exit(1)
		}
	}

	if err := k.mount(SourceTmpfsRootfs, intermediateHostPath, FstypeTmpfs, MS_NODEV|MS_NOSUID, zeroString); err != nil {
		k.fatalf("cannot mount intermediate root: %v", err)
	}
	if err := k.chdir(intermediateHostPath); err != nil {
		k.fatalf("cannot enter intermediate host path: %v", err)
	}

	if err := k.mkdir(sysrootDir, 0755); err != nil {
		k.fatalf("%v", err)
	}
	if err := k.mount(sysrootDir, sysrootDir, zeroString, MS_SILENT|MS_BIND|MS_REC, zeroString); err != nil {
		k.fatalf("cannot bind sysroot: %v", err)
	}

	if err := k.mkdir(hostDir, 0755); err != nil {
		k.fatalf("%v", err)
	}
	// pivot_root uncovers intermediateHostPath in hostDir
	if err := k.pivotRoot(intermediateHostPath, hostDir); err != nil {
		k.fatalf("cannot pivot into intermediate root: %v", err)
	}
	if err := k.chdir(FHSRoot); err != nil {
		k.fatalf("cannot enter intermediate root: %v", err)
	}

	/* apply is called right after pivot_root and entering the new root;
	this step sets up the container filesystem, and implementations are expected to keep the host root
	and sysroot mount points intact but otherwise can do whatever they need to;
	chdir is allowed but discouraged */
	for i, op := range *params.Ops {
		// ops already checked during early setup
		k.verbosef("%s %s", op.prefix(), op)
		if err := op.apply(state, k); err != nil {
			k.printBaseErr(err,
				fmt.Sprintf("cannot apply op at index %d:", i))
			k.beforeExit()
			k.exit(1)
		}
	}

	// setup requiring host root complete at this point
	if err := k.mount(hostDir, hostDir, zeroString, MS_SILENT|MS_REC|MS_PRIVATE, zeroString); err != nil {
		k.fatalf("cannot make host root rprivate: %v", err)
	}
	if err := k.unmount(hostDir, MNT_DETACH); err != nil {
		k.fatalf("cannot unmount host root: %v", err)
	}

	{
		var fd int
		if err := IgnoringEINTR(func() (err error) {
			fd, err = k.open(FHSRoot, O_DIRECTORY|O_RDONLY, 0)
			return
		}); err != nil {
			k.fatalf("cannot open intermediate root: %v", err)
		}
		if err := k.chdir(sysrootPath); err != nil {
			k.fatalf("cannot enter sysroot: %v", err)
		}

		if err := k.pivotRoot(".", "."); err != nil {
			k.fatalf("cannot pivot into sysroot: %v", err)
		}
		if err := k.fchdir(fd); err != nil {
			k.fatalf("cannot re-enter intermediate root: %v", err)
		}
		if err := k.unmount(".", MNT_DETACH); err != nil {
			k.fatalf("cannot unmount intemediate root: %v", err)
		}
		if err := k.chdir(FHSRoot); err != nil {
			k.fatalf("cannot enter root: %v", err)
		}

		if err := k.close(fd); err != nil {
			k.fatalf("cannot close intermediate root: %v", err)
		}
	}

	if err := k.capAmbientClearAll(); err != nil {
		k.fatalf("cannot clear the ambient capability set: %v", err)
	}
	for i := uintptr(0); i <= k.lastcap(); i++ {
		if params.Privileged && i == CAP_SYS_ADMIN {
			continue
		}
		if err := k.capBoundingSetDrop(i); err != nil {
			k.fatalf("cannot drop capability from bounding set: %v", err)
		}
	}

	var keep [2]uint32
	if params.Privileged {
		keep[capToIndex(CAP_SYS_ADMIN)] |= capToMask(CAP_SYS_ADMIN)

		if err := k.capAmbientRaise(CAP_SYS_ADMIN); err != nil {
			k.fatalf("cannot raise CAP_SYS_ADMIN: %v", err)
		}
	}
	if err := k.capset(
		&capHeader{_LINUX_CAPABILITY_VERSION_3, 0},
		&[2]capData{{0, keep[0], keep[0]}, {0, keep[1], keep[1]}},
	); err != nil {
		k.fatalf("cannot capset: %v", err)
	}

	if !params.SeccompDisable {
		rules := params.SeccompRules
		if len(rules) == 0 { // non-empty rules slice always overrides presets
			k.verbosef("resolving presets %#x", params.SeccompPresets)
			rules = seccomp.Preset(params.SeccompPresets, params.SeccompFlags)
		}
		if err := k.seccompLoad(rules, params.SeccompFlags); err != nil {
			// this also indirectly asserts PR_SET_NO_NEW_PRIVS
			k.fatalf("cannot load syscall filter: %v", err)
		}
		k.verbosef("%d filter rules loaded", len(rules))
	} else {
		k.verbose("syscall filter not configured")
	}

	extraFiles := make([]*os.File, params.Count)
	for i := range extraFiles {
		// setup fd is placed before all extra files
		extraFiles[i] = k.newFile(uintptr(offsetSetup+i), "extra file "+strconv.Itoa(i))
	}
	k.umask(oldmask)

	cmd := exec.Command(params.Path.String())
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Args = params.Args
	cmd.Env = params.Env
	cmd.ExtraFiles = extraFiles
	cmd.Dir = params.Dir.String()

	k.verbosef("starting initial program %s", params.Path)
	if err := k.start(cmd); err != nil {
		k.fatalf("%v", err)
	}
	k.suspend()

	if err := closeSetup(); err != nil {
		k.printf("cannot close setup pipe: %v", err)
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
				wpid, err = k.wait4(-1, &wstatus, 0, nil)
			}
		}
		if !errors.Is(err, ECHILD) {
			k.printf("unexpected wait4 response: %v", err)
		}

		close(done)
	}()

	// handle signals to dump withheld messages
	sig := make(chan os.Signal, 2)
	k.notify(sig, os.Interrupt, CancelSignal)

	// closed after residualProcessTimeout has elapsed after initial process death
	timeout := make(chan struct{})

	r := 2
	for {
		select {
		case s := <-sig:
			if k.resume() {
				k.verbosef("%s after process start", s.String())
			} else {
				k.verbosef("got %s", s.String())
			}
			if s == CancelSignal && params.ForwardCancel && cmd.Process != nil {
				k.verbose("forwarding context cancellation")
				if err := k.signal(cmd, os.Interrupt); err != nil {
					k.printf("cannot forward cancellation: %v", err)
				}
				continue
			}
			k.exit(0)

		case w := <-info:
			if w.wpid == cmd.Process.Pid {
				// initial process exited, output is most likely available again
				k.resume()

				switch {
				case w.wstatus.Exited():
					r = w.wstatus.ExitStatus()
					k.verbosef("initial process exited with code %d", w.wstatus.ExitStatus())
				case w.wstatus.Signaled():
					r = 128 + int(w.wstatus.Signal())
					k.verbosef("initial process exited with signal %s", w.wstatus.Signal())
				default:
					r = 255
					k.verbosef("initial process exited with status %#x", w.wstatus)
				}

				go func() { time.Sleep(params.AdoptWaitDelay); close(timeout) }()
			}

		case <-done:
			k.beforeExit()
			k.exit(r)

		case <-timeout:
			k.printf("timeout exceeded waiting for lingering processes")
			k.beforeExit()
			k.exit(r)
		}
	}
}

const initName = "init"

// TryArgv0 calls [Init] if the last element of argv0 is "init".
func TryArgv0(v Msg, prepare func(prefix string), setVerbose func(verbose bool)) {
	if len(os.Args) > 0 && path.Base(os.Args[0]) == initName {
		msg = v
		Init(prepare, setVerbose)
		msg.BeforeExit()
		os.Exit(0)
	}
}
