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
	"path"
	"strconv"
	. "syscall"
	"time"

	"hakurei.app/container/seccomp"
)

const (
	// Nonexistent is a path that cannot exist.
	// /proc is chosen because a system with covered /proc is unsupported by this package.
	Nonexistent = "/proc/nonexistent"

	// CancelSignal is the signal expected by container init on context cancel.
	// A custom [Container.Cancel] function must eventually deliver this signal.
	CancelSignal = SIGTERM
)

type (
	// Container represents a container environment being prepared or run.
	// None of [Container] methods are safe for concurrent use.
	Container struct {
		// Name of initial process in the container.
		name string
		// Cgroup fd, nil to disable.
		Cgroup *int
		// ExtraFiles passed through to initial process in the container,
		// with behaviour identical to its [exec.Cmd] counterpart.
		ExtraFiles []*os.File

		// param encoder for shim and init
		setup *gob.Encoder
		// cancels cmd
		cancel context.CancelFunc

		Stdin  io.Reader
		Stdout io.Writer
		Stderr io.Writer

		Cancel    func(cmd *exec.Cmd) error
		WaitDelay time.Duration

		cmd *exec.Cmd
		ctx context.Context
		Params
	}

	// Params holds container configuration and is safe to serialise.
	Params struct {
		// Working directory in the container.
		Dir string
		// Initial process environment.
		Env []string
		// Absolute path of initial process in the container. Overrides name.
		Path string
		// Initial process argv.
		Args []string
		// Deliver SIGINT to the initial process on context cancellation.
		ForwardCancel bool
		// time to wait for linger processes after death of initial process
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
		SeccompPresets seccomp.FilterPreset
		// Do not load seccomp program.
		SeccompDisable bool

		// Permission bits of newly created parent directories.
		// The zero value is interpreted as 0755.
		ParentPerm os.FileMode
		// Do not syscall.Setsid.
		RetainSession bool
		// Do not [syscall.CLONE_NEWNET].
		HostNet bool
		// Retain CAP_SYS_ADMIN.
		Privileged bool
	}
)

// Start starts the container init. The init process blocks until Serve is called.
func (p *Container) Start() error {
	if p.cmd != nil {
		return errors.New("container: already started")
	}
	if p.Ops == nil || len(*p.Ops) == 0 {
		return errors.New("container: starting an empty container")
	}

	ctx, cancel := context.WithCancel(p.ctx)
	p.cancel = cancel

	// map to overflow id to work around ownership checks
	if p.Uid < 1 {
		p.Uid = OverflowUid()
	}
	if p.Gid < 1 {
		p.Gid = OverflowGid()
	}

	if !p.RetainSession {
		p.SeccompPresets |= seccomp.PresetDenyTTY
	}

	if p.AdoptWaitDelay == 0 {
		p.AdoptWaitDelay = 5 * time.Second
	}
	// to allow disabling this behaviour
	if p.AdoptWaitDelay < 0 {
		p.AdoptWaitDelay = 0
	}

	p.cmd = exec.CommandContext(ctx, MustExecutable())
	p.cmd.Args = []string{initName}
	p.cmd.Stdin, p.cmd.Stdout, p.cmd.Stderr = p.Stdin, p.Stdout, p.Stderr
	p.cmd.WaitDelay = p.WaitDelay
	if p.Cancel != nil {
		p.cmd.Cancel = func() error { return p.Cancel(p.cmd) }
	} else {
		p.cmd.Cancel = func() error { return p.cmd.Process.Signal(CancelSignal) }
	}
	p.cmd.Dir = "/"
	p.cmd.SysProcAttr = &SysProcAttr{
		Setsid:    !p.RetainSession,
		Pdeathsig: SIGKILL,
		Cloneflags: CLONE_NEWUSER | CLONE_NEWPID | CLONE_NEWNS |
			CLONE_NEWIPC | CLONE_NEWUTS | CLONE_NEWCGROUP,

		// remain privileged for setup
		AmbientCaps: []uintptr{CAP_SYS_ADMIN, CAP_SETPCAP},

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
		return wrapErrSuffix(err,
			"cannot create shim setup pipe:")
	} else {
		p.setup = e
		p.cmd.Env = []string{setupEnv + "=" + strconv.Itoa(fd)}
	}
	p.cmd.ExtraFiles = append(p.cmd.ExtraFiles, p.ExtraFiles...)

	msg.Verbose("starting container init")
	if err := p.cmd.Start(); err != nil {
		return msg.WrapErr(err, err.Error())
	}
	return nil
}

// Serve serves [Container.Params] to the container init.
// Serve must only be called once.
func (p *Container) Serve() error {
	if p.setup == nil {
		panic("invalid serve")
	}

	setup := p.setup
	p.setup = nil

	if p.Path != "" && !path.IsAbs(p.Path) {
		p.cancel()
		return msg.WrapErr(EINVAL,
			fmt.Sprintf("invalid executable path %q", p.Path))
	}

	if p.Path == "" {
		if p.name == "" {
			p.Path = os.Getenv("SHELL")
			if !path.IsAbs(p.Path) {
				p.cancel()
				return msg.WrapErr(EBADE,
					"no command specified and $SHELL is invalid")
			}
			p.name = path.Base(p.Path)
		} else if path.IsAbs(p.name) {
			p.Path = p.name
		} else if v, err := exec.LookPath(p.name); err != nil {
			p.cancel()
			return msg.WrapErr(err, err.Error())
		} else {
			p.Path = v
		}
	}

	if p.SeccompRules == nil {
		// do not transmit nil
		p.SeccompRules = make([]seccomp.NativeRule, 0)
	}

	err := setup.Encode(
		&initParams{
			p.Params,
			Getuid(),
			Getgid(),
			len(p.ExtraFiles),
			msg.IsVerbose(),
		},
	)
	if err != nil {
		p.cancel()
	}
	return err
}

// Wait waits for the container init process to exit.
func (p *Container) Wait() error { defer p.cancel(); return p.cmd.Wait() }

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

func New(ctx context.Context, name string, args ...string) *Container {
	return &Container{name: name, ctx: ctx,
		Params: Params{Args: append([]string{name}, args...), Dir: "/", Ops: new(Ops)},
	}
}
