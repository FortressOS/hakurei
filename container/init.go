package container

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"slices"
	"strconv"
	. "syscall"
	"time"

	"hakurei.app/container/fhs"
	"hakurei.app/container/seccomp"
	"hakurei.app/message"
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
	intermediateHostPath = fhs.Proc + "self/fd"

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

		// prefix returns a log message prefix, and whether this Op prints no identifying message on its own.
		prefix() (string, bool)

		Is(op Op) bool
		Valid() bool
		fmt.Stringer
	}

	// setupState persists context between Ops.
	setupState struct {
		nonrepeatable uintptr
		*Params
		message.Msg
	}
)

// Grow grows the slice Ops points to using [slices.Grow].
func (f *Ops) Grow(n int) { *f = slices.Grow(*f, n) }

const (
	nrAutoEtc = 1 << iota
	nrAutoRoot
)

// OpRepeatError is returned applying a repeated nonrepeatable [Op].
type OpRepeatError string

func (e OpRepeatError) Error() string { return string(e) + " is not repeatable" }

// OpStateError indicates an impossible internal state has been reached in an [Op].
type OpStateError string

func (o OpStateError) Error() string { return "impossible " + string(o) + " state reached" }

// initParams are params passed from parent.
type initParams struct {
	Params

	HostUid, HostGid int
	// extra files count
	Count int
	// verbosity pass through
	Verbose bool
}

// Init is called by [TryArgv0] if the current process is the container init.
func Init(msg message.Msg) { initEntrypoint(direct{}, msg) }

func initEntrypoint(k syscallDispatcher, msg message.Msg) {
	k.lockOSThread()

	if msg == nil {
		panic("attempting to call initEntrypoint with nil msg")
	}

	if k.getpid() != 1 {
		k.fatal(msg, "this process must run as pid 1")
	}

	if err := k.setPtracer(0); err != nil {
		msg.Verbosef("cannot enable ptrace protection via Yama LSM: %v", err)
		// not fatal: this program has no additional privileges at initial program start
	}

	var (
		params      initParams
		closeSetup  func() error
		setupFd     uintptr
		offsetSetup int
	)
	if f, err := k.receive(setupEnv, &params, &setupFd); err != nil {
		if errors.Is(err, EBADF) {
			k.fatal(msg, "invalid setup descriptor")
		}
		if errors.Is(err, ErrReceiveEnv) {
			k.fatal(msg, setupEnv+" not set")
		}

		k.fatalf(msg, "cannot decode init setup payload: %v", err)
	} else {
		if params.Ops == nil {
			k.fatal(msg, "invalid setup parameters")
		}
		if params.ParentPerm == 0 {
			params.ParentPerm = 0755
		}

		msg.SwapVerbose(params.Verbose)
		msg.Verbose("received setup parameters")
		closeSetup = f
		offsetSetup = int(setupFd + 1)
	}

	// write uid/gid map here so parent does not need to set dumpable
	if err := k.setDumpable(SUID_DUMP_USER); err != nil {
		k.fatalf(msg, "cannot set SUID_DUMP_USER: %v", err)
	}
	if err := k.writeFile(fhs.Proc+"self/uid_map",
		append([]byte{}, strconv.Itoa(params.Uid)+" "+strconv.Itoa(params.HostUid)+" 1\n"...),
		0); err != nil {
		k.fatalf(msg, "%v", err)
	}
	if err := k.writeFile(fhs.Proc+"self/setgroups",
		[]byte("deny\n"),
		0); err != nil && !os.IsNotExist(err) {
		k.fatalf(msg, "%v", err)
	}
	if err := k.writeFile(fhs.Proc+"self/gid_map",
		append([]byte{}, strconv.Itoa(params.Gid)+" "+strconv.Itoa(params.HostGid)+" 1\n"...),
		0); err != nil {
		k.fatalf(msg, "%v", err)
	}
	if err := k.setDumpable(SUID_DUMP_DISABLE); err != nil {
		k.fatalf(msg, "cannot set SUID_DUMP_DISABLE: %v", err)
	}

	oldmask := k.umask(0)
	if params.Hostname != "" {
		if err := k.sethostname([]byte(params.Hostname)); err != nil {
			k.fatalf(msg, "cannot set hostname: %v", err)
		}
	}

	// cache sysctl before pivot_root
	lastcap := k.lastcap(msg)

	if err := k.mount(zeroString, fhs.Root, zeroString, MS_SILENT|MS_SLAVE|MS_REC, zeroString); err != nil {
		k.fatalf(msg, "cannot make / rslave: %v", optionalErrorUnwrap(err))
	}

	state := &setupState{Params: &params.Params, Msg: msg}

	/* early is called right before pivot_root into intermediate root;
	this step is mostly for gathering information that would otherwise be difficult to obtain
	via library functions after pivot_root, and implementations are expected to avoid changing
	the state of the mount namespace */
	for i, op := range *params.Ops {
		if op == nil || !op.Valid() {
			k.fatalf(msg, "invalid op at index %d", i)
		}

		if err := op.early(state, k); err != nil {
			if m, ok := messageFromError(err); ok {
				k.fatal(msg, m)
			} else {
				k.fatalf(msg, "cannot prepare op at index %d: %v", i, err)
			}
		}
	}

	if err := k.mount(SourceTmpfsRootfs, intermediateHostPath, FstypeTmpfs, MS_NODEV|MS_NOSUID, zeroString); err != nil {
		k.fatalf(msg, "cannot mount intermediate root: %v", optionalErrorUnwrap(err))
	}
	if err := k.chdir(intermediateHostPath); err != nil {
		k.fatalf(msg, "cannot enter intermediate host path: %v", err)
	}

	if err := k.mkdir(sysrootDir, 0755); err != nil {
		k.fatalf(msg, "%v", err)
	}
	if err := k.mount(sysrootDir, sysrootDir, zeroString, MS_SILENT|MS_BIND|MS_REC, zeroString); err != nil {
		k.fatalf(msg, "cannot bind sysroot: %v", optionalErrorUnwrap(err))
	}

	if err := k.mkdir(hostDir, 0755); err != nil {
		k.fatalf(msg, "%v", err)
	}
	// pivot_root uncovers intermediateHostPath in hostDir
	if err := k.pivotRoot(intermediateHostPath, hostDir); err != nil {
		k.fatalf(msg, "cannot pivot into intermediate root: %v", err)
	}
	if err := k.chdir(fhs.Root); err != nil {
		k.fatalf(msg, "cannot enter intermediate root: %v", err)
	}

	/* apply is called right after pivot_root and entering the new root;
	this step sets up the container filesystem, and implementations are expected to keep the host root
	and sysroot mount points intact but otherwise can do whatever they need to;
	chdir is allowed but discouraged */
	for i, op := range *params.Ops {
		// ops already checked during early setup
		if prefix, ok := op.prefix(); ok {
			msg.Verbosef("%s %s", prefix, op)
		}
		if err := op.apply(state, k); err != nil {
			if m, ok := messageFromError(err); ok {
				k.fatal(msg, m)
			} else {
				k.fatalf(msg, "cannot apply op at index %d: %v", i, err)
			}
		}
	}

	// setup requiring host root complete at this point
	if err := k.mount(hostDir, hostDir, zeroString, MS_SILENT|MS_REC|MS_PRIVATE, zeroString); err != nil {
		k.fatalf(msg, "cannot make host root rprivate: %v", optionalErrorUnwrap(err))
	}
	if err := k.unmount(hostDir, MNT_DETACH); err != nil {
		k.fatalf(msg, "cannot unmount host root: %v", err)
	}

	{
		var fd int
		if err := IgnoringEINTR(func() (err error) {
			fd, err = k.open(fhs.Root, O_DIRECTORY|O_RDONLY, 0)
			return
		}); err != nil {
			k.fatalf(msg, "cannot open intermediate root: %v", err)
		}
		if err := k.chdir(sysrootPath); err != nil {
			k.fatalf(msg, "cannot enter sysroot: %v", err)
		}

		if err := k.pivotRoot(".", "."); err != nil {
			k.fatalf(msg, "cannot pivot into sysroot: %v", err)
		}
		if err := k.fchdir(fd); err != nil {
			k.fatalf(msg, "cannot re-enter intermediate root: %v", err)
		}
		if err := k.unmount(".", MNT_DETACH); err != nil {
			k.fatalf(msg, "cannot unmount intermediate root: %v", err)
		}
		if err := k.chdir(fhs.Root); err != nil {
			k.fatalf(msg, "cannot enter root: %v", err)
		}

		if err := k.close(fd); err != nil {
			k.fatalf(msg, "cannot close intermediate root: %v", err)
		}
	}

	if err := k.capAmbientClearAll(); err != nil {
		k.fatalf(msg, "cannot clear the ambient capability set: %v", err)
	}
	for i := uintptr(0); i <= lastcap; i++ {
		if params.Privileged && i == CAP_SYS_ADMIN {
			continue
		}
		if err := k.capBoundingSetDrop(i); err != nil {
			k.fatalf(msg, "cannot drop capability from bounding set: %v", err)
		}
	}

	var keep [2]uint32
	if params.Privileged {
		keep[capToIndex(CAP_SYS_ADMIN)] |= capToMask(CAP_SYS_ADMIN)

		if err := k.capAmbientRaise(CAP_SYS_ADMIN); err != nil {
			k.fatalf(msg, "cannot raise CAP_SYS_ADMIN: %v", err)
		}
	}
	if err := k.capset(
		&capHeader{_LINUX_CAPABILITY_VERSION_3, 0},
		&[2]capData{{0, keep[0], keep[0]}, {0, keep[1], keep[1]}},
	); err != nil {
		k.fatalf(msg, "cannot capset: %v", err)
	}

	if !params.SeccompDisable {
		rules := params.SeccompRules
		if len(rules) == 0 { // non-empty rules slice always overrides presets
			msg.Verbosef("resolving presets %#x", params.SeccompPresets)
			rules = seccomp.Preset(params.SeccompPresets, params.SeccompFlags)
		}
		if err := k.seccompLoad(rules, params.SeccompFlags); err != nil {
			// this also indirectly asserts PR_SET_NO_NEW_PRIVS
			k.fatalf(msg, "cannot load syscall filter: %v", err)
		}
		msg.Verbosef("%d filter rules loaded", len(rules))
	} else {
		msg.Verbose("syscall filter not configured")
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

	msg.Verbosef("starting initial program %s", params.Path)
	if err := k.start(cmd); err != nil {
		k.fatalf(msg, "%v", err)
	}

	if err := closeSetup(); err != nil {
		k.printf(msg, "cannot close setup pipe: %v", err)
		// not fatal
	}

	type winfo struct {
		wpid    int
		wstatus WaitStatus
	}

	// info is closed as the wait4 thread terminates
	// when there are no longer any processes left to reap
	info := make(chan winfo, 1)

	k.new(func(k syscallDispatcher) {
		k.lockOSThread()

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
			k.printf(msg, "unexpected wait4 response: %v", err)
		}

		close(info)
	})

	// handle signals to dump withheld messages
	sig := make(chan os.Signal, 2)
	k.notify(sig, CancelSignal,
		os.Interrupt, SIGTERM, SIGQUIT)

	// closed after residualProcessTimeout has elapsed after initial process death
	timeout := make(chan struct{})

	r := 2
	for {
		select {
		case s := <-sig:
			if s == CancelSignal && params.ForwardCancel && cmd.Process != nil {
				msg.Verbose("forwarding context cancellation")
				if err := k.signal(cmd, os.Interrupt); err != nil {
					k.printf(msg, "cannot forward cancellation: %v", err)
				}
				continue
			}

			if s == SIGTERM || s == SIGQUIT {
				msg.Verbosef("got %s, forwarding to initial process", s.String())
				if err := k.signal(cmd, s); err != nil {
					k.printf(msg, "cannot forward signal: %v", err)
				}
				continue
			}

			msg.Verbosef("got %s", s.String())
			msg.BeforeExit()
			k.exit(0)

		case w, ok := <-info:
			if !ok {
				msg.BeforeExit()
				k.exit(r)
				continue // unreachable
			}

			if w.wpid == cmd.Process.Pid {
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

				go func() { time.Sleep(params.AdoptWaitDelay); close(timeout) }()
			}

		case <-timeout:
			k.printf(msg, "timeout exceeded waiting for lingering processes")
			msg.BeforeExit()
			k.exit(r)
		}
	}
}

// initName is the prefix used by log.std in the init process.
const initName = "init"

// TryArgv0 calls [Init] if the last element of argv0 is "init".
// If a nil msg is passed, the system logger is used instead.
func TryArgv0(msg message.Msg) {
	if msg == nil {
		log.SetPrefix(initName + ": ")
		log.SetFlags(0)
		msg = message.New(log.Default())
	}

	if len(os.Args) > 0 && path.Base(os.Args[0]) == initName {
		Init(msg)
		msg.BeforeExit()
		os.Exit(0)
	}
}
