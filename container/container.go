// Package container implements unprivileged Linux containers with built-in support for syscall filtering.
package container

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	. "syscall"
	"time"

	"hakurei.app/container/bits"
	"hakurei.app/container/seccomp"
)

const (
	// CancelSignal is the signal expected by container init on context cancel.
	// A custom [Container.Cancel] function must eventually deliver this signal.
	CancelSignal = SIGTERM
)

type (
	// Container represents a container environment being prepared or run.
	// None of [Container] methods are safe for concurrent use.
	Container struct {
		// Cgroup fd, nil to disable.
		Cgroup *int
		// ExtraFiles passed through to initial process in the container,
		// with behaviour identical to its [exec.Cmd] counterpart.
		ExtraFiles []*os.File

		// param encoder for shim and init
		setup *gob.Encoder
		// cancels cmd
		cancel context.CancelFunc
		// closed after Wait returns
		wait chan struct{}

		Stdin  io.Reader
		Stdout io.Writer
		Stderr io.Writer

		Cancel    func(cmd *exec.Cmd) error
		WaitDelay time.Duration

		cmd *exec.Cmd
		ctx context.Context
		msg Msg
		Params
	}

	// Params holds container configuration and is safe to serialise.
	Params struct {
		// Working directory in the container.
		Dir *Absolute
		// Initial process environment.
		Env []string
		// Pathname of initial process in the container.
		Path *Absolute
		// Initial process argv.
		Args []string
		// Deliver SIGINT to the initial process on context cancellation.
		ForwardCancel bool
		// Time to wait for processes lingering after the initial process terminates.
		AdoptWaitDelay time.Duration

		// Mapped Uid in user namespace.
		Uid int
		// Mapped Gid in user namespace.
		Gid int
		// Hostname value in UTS namespace.
		Hostname string
		// Sequential container setup ops.
		*Ops

		// Seccomp system call filter rules.
		SeccompRules []seccomp.NativeRule
		// Extra seccomp flags.
		SeccompFlags seccomp.ExportFlag
		// Seccomp presets. Has no effect unless SeccompRules is zero-length.
		SeccompPresets bits.FilterPreset
		// Do not load seccomp program.
		SeccompDisable bool

		// Permission bits of newly created parent directories.
		// The zero value is interpreted as 0755.
		ParentPerm os.FileMode
		// Do not syscall.Setsid.
		RetainSession bool
		// Do not [syscall.CLONE_NEWNET].
		HostNet bool
		// Do not [LANDLOCK_SCOPE_ABSTRACT_UNIX_SOCKET].
		HostAbstract bool
		// Retain CAP_SYS_ADMIN.
		Privileged bool
	}
)

// A StartError contains additional information on a container startup failure.
type StartError struct {
	// Fatal suggests whether this error should be considered fatal for the entire program.
	Fatal bool
	// Step refers to the part of the setup this error is returned from.
	Step string
	// Err is the underlying error.
	Err error
	// Origin is whether this error originated from the [Container.Start] method.
	Origin bool
	// Passthrough is whether the Error method is passed through to Err.
	Passthrough bool
}

func (e *StartError) Unwrap() error { return e.Err }
func (e *StartError) Error() string {
	if e.Passthrough {
		return e.Err.Error()
	}
	if e.Origin {
		return e.Step
	}

	{
		var syscallError *os.SyscallError
		if errors.As(e.Err, &syscallError) && syscallError != nil {
			return e.Step + " " + syscallError.Error()
		}
	}

	return e.Step + ": " + e.Err.Error()
}

// Message returns a user-facing error message.
func (e *StartError) Message() string {
	if e.Passthrough {
		switch {
		case errors.As(e.Err, new(*os.PathError)),
			errors.As(e.Err, new(*os.SyscallError)):
			return "cannot " + e.Err.Error()

		default:
			return e.Err.Error()
		}
	}
	if e.Origin {
		return e.Step
	}
	return "cannot " + e.Error()
}

// Start starts the container init. The init process blocks until Serve is called.
func (p *Container) Start() error {
	if p == nil || p.cmd == nil ||
		p.Ops == nil || len(*p.Ops) == 0 {
		return errors.New("container: starting an invalid container")
	}
	if p.cmd.Process != nil {
		return errors.New("container: already started")
	}

	// map to overflow id to work around ownership checks
	if p.Uid < 1 {
		p.Uid = OverflowUid(p.msg)
	}
	if p.Gid < 1 {
		p.Gid = OverflowGid(p.msg)
	}

	if !p.RetainSession {
		p.SeccompPresets |= bits.PresetDenyTTY
	}

	if p.AdoptWaitDelay == 0 {
		p.AdoptWaitDelay = 5 * time.Second
	}
	// to allow disabling this behaviour
	if p.AdoptWaitDelay < 0 {
		p.AdoptWaitDelay = 0
	}

	if p.cmd.Stdin == nil {
		p.cmd.Stdin = p.Stdin
	}
	if p.cmd.Stdout == nil {
		p.cmd.Stdout = p.Stdout
	}
	if p.cmd.Stderr == nil {
		p.cmd.Stderr = p.Stderr
	}

	p.cmd.Args = []string{initName}
	p.cmd.WaitDelay = p.WaitDelay
	if p.Cancel != nil {
		p.cmd.Cancel = func() error { return p.Cancel(p.cmd) }
	} else {
		p.cmd.Cancel = func() error { return p.cmd.Process.Signal(CancelSignal) }
	}
	p.cmd.Dir = FHSRoot
	p.cmd.SysProcAttr = &SysProcAttr{
		Setsid:    !p.RetainSession,
		Pdeathsig: SIGKILL,
		Cloneflags: CLONE_NEWUSER | CLONE_NEWPID | CLONE_NEWNS |
			CLONE_NEWIPC | CLONE_NEWUTS | CLONE_NEWCGROUP,

		AmbientCaps: []uintptr{
			// general container setup
			CAP_SYS_ADMIN,
			// drop capabilities
			CAP_SETPCAP,
			// overlay access to upperdir and workdir
			CAP_DAC_OVERRIDE,
		},

		UseCgroupFD: p.Cgroup != nil,
	}
	if p.cmd.SysProcAttr.UseCgroupFD {
		p.cmd.SysProcAttr.CgroupFD = *p.Cgroup
	}
	if !p.HostNet {
		p.cmd.SysProcAttr.Cloneflags |= CLONE_NEWNET
	}

	// place setup pipe before user supplied extra files, this is later restored by init
	if fd, e, err := Setup(&p.cmd.ExtraFiles); err != nil {
		return &StartError{true, "set up params stream", err, false, false}
	} else {
		p.setup = e
		p.cmd.Env = []string{setupEnv + "=" + strconv.Itoa(fd)}
	}
	p.cmd.ExtraFiles = append(p.cmd.ExtraFiles, p.ExtraFiles...)

	done := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		p.wait = make(chan struct{})

		done <- func() error { // setup depending on per-thread state must happen here
			// PR_SET_NO_NEW_PRIVS: depends on per-thread state but acts on all processes created from that thread
			if err := SetNoNewPrivs(); err != nil {
				return &StartError{true, "prctl(PR_SET_NO_NEW_PRIVS)", err, false, false}
			}

			// landlock: depends on per-thread state but acts on a process group
			{
				rulesetAttr := &RulesetAttr{Scoped: LANDLOCK_SCOPE_SIGNAL}
				if !p.HostAbstract {
					rulesetAttr.Scoped |= LANDLOCK_SCOPE_ABSTRACT_UNIX_SOCKET
				}

				if abi, err := LandlockGetABI(); err != nil {
					if p.HostAbstract {
						// landlock can be skipped here as it restricts access to resources
						// already covered by namespaces (pid)
						goto landlockOut
					}
					return &StartError{false, "get landlock ABI", err, false, false}
				} else if abi < 6 {
					if p.HostAbstract {
						// see above comment
						goto landlockOut
					}
					return &StartError{false, "kernel version too old for LANDLOCK_SCOPE_ABSTRACT_UNIX_SOCKET", ENOSYS, true, false}
				} else {
					p.msg.Verbosef("landlock abi version %d", abi)
				}

				if rulesetFd, err := rulesetAttr.Create(0); err != nil {
					return &StartError{true, "create landlock ruleset", err, false, false}
				} else {
					p.msg.Verbosef("enforcing landlock ruleset %s", rulesetAttr)
					if err = LandlockRestrictSelf(rulesetFd, 0); err != nil {
						_ = Close(rulesetFd)
						return &StartError{true, "enforce landlock ruleset", err, false, false}
					}
					if err = Close(rulesetFd); err != nil {
						p.msg.Verbosef("cannot close landlock ruleset: %v", err)
						// not fatal
					}
				}

			landlockOut:
			}

			p.msg.Verbose("starting container init")
			if err := p.cmd.Start(); err != nil {
				return &StartError{false, "start container init", err, false, true}
			}
			return nil
		}()

		// keep this thread alive until Wait returns for cancel
		<-p.wait
	}()
	return <-done
}

// Serve serves [Container.Params] to the container init.
// Serve must only be called once.
func (p *Container) Serve() error {
	if p.setup == nil {
		panic("invalid serve")
	}

	setup := p.setup
	p.setup = nil

	if p.Path == nil {
		p.cancel()
		return &StartError{false, "invalid executable pathname", EINVAL, true, false}
	}

	// do not transmit nil
	if p.Dir == nil {
		p.Dir = AbsFHSRoot
	}
	if p.SeccompRules == nil {
		p.SeccompRules = make([]seccomp.NativeRule, 0)
	}

	err := setup.Encode(
		&initParams{
			p.Params,
			Getuid(),
			Getgid(),
			len(p.ExtraFiles),
			p.msg.IsVerbose(),
		},
	)
	if err != nil {
		p.cancel()
	}
	return err
}

// Wait waits for the container init process to exit and releases any resources associated with the [Container].
func (p *Container) Wait() error {
	if p.cmd == nil || p.cmd.Process == nil {
		return EINVAL
	}

	err := p.cmd.Wait()
	p.cancel()
	if p.wait != nil && err == nil {
		close(p.wait)
	}
	return err
}

// StdinPipe calls the [exec.Cmd] method with the same name.
func (p *Container) StdinPipe() (w io.WriteCloser, err error) {
	if p.Stdin != nil {
		return nil, errors.New("container: Stdin already set")
	}
	w, err = p.cmd.StdinPipe()
	p.Stdin = p.cmd.Stdin
	return
}

// StdoutPipe calls the [exec.Cmd] method with the same name.
func (p *Container) StdoutPipe() (r io.ReadCloser, err error) {
	if p.Stdout != nil {
		return nil, errors.New("container: Stdout already set")
	}
	r, err = p.cmd.StdoutPipe()
	p.Stdout = p.cmd.Stdout
	return
}

// StderrPipe calls the [exec.Cmd] method with the same name.
func (p *Container) StderrPipe() (r io.ReadCloser, err error) {
	if p.Stderr != nil {
		return nil, errors.New("container: Stderr already set")
	}
	r, err = p.cmd.StderrPipe()
	p.Stderr = p.cmd.Stderr
	return
}

func (p *Container) String() string {
	return fmt.Sprintf("argv: %q, filter: %v, rules: %d, flags: %#x, presets: %#x",
		p.Args, !p.SeccompDisable, len(p.SeccompRules), int(p.SeccompFlags), int(p.SeccompPresets))
}

// ProcessState returns the address to os.ProcessState held by the underlying [exec.Cmd].
func (p *Container) ProcessState() *os.ProcessState {
	if p.cmd == nil {
		return nil
	}
	return p.cmd.ProcessState
}

// New returns the address to a new instance of [Container] that requires further initialisation before use.
func New(ctx context.Context, msg Msg) *Container {
	if msg == nil {
		msg = NewMsg(nil)
	}

	p := &Container{ctx: ctx, msg: msg, Params: Params{Ops: new(Ops)}}
	c, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.cmd = exec.CommandContext(c, MustExecutable(msg))
	return p
}

// NewCommand calls [New] and initialises the [Params.Path] and [Params.Args] fields.
func NewCommand(ctx context.Context, msg Msg, pathname *Absolute, name string, args ...string) *Container {
	z := New(ctx, msg)
	z.Path = pathname
	z.Args = append([]string{name}, args...)
	return z
}
