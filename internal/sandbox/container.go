package sandbox

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
	"syscall"
	"time"

	"git.gensokyo.uk/security/fortify/helper/proc"
	"git.gensokyo.uk/security/fortify/seccomp"
)

type HardeningFlags uintptr

const (
	FSyscallCompat HardeningFlags = 1 << iota
	FAllowDevel
	FAllowUserns
	FAllowTTY
	FAllowNet
)

func (flags HardeningFlags) seccomp(opts seccomp.SyscallOpts) seccomp.SyscallOpts {
	if flags&FSyscallCompat == 0 {
		opts |= seccomp.FlagExt
	}
	if flags&FAllowDevel == 0 {
		opts |= seccomp.FlagDenyDevel
	}
	if flags&FAllowUserns == 0 {
		opts |= seccomp.FlagDenyNS
	}
	if flags&FAllowTTY == 0 {
		opts |= seccomp.FlagDenyTTY
	}
	return opts
}

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

		InitParams
		// Custom [exec.Cmd] initialisation function.
		CommandContext func(ctx context.Context) (cmd *exec.Cmd)

		// param encoder for shim and init
		setup *gob.Encoder
		// cancels cmd
		cancel context.CancelFunc

		Stdin  io.Reader
		Stdout io.Writer
		Stderr io.Writer

		Cancel    func() error
		WaitDelay time.Duration

		cmd *exec.Cmd
		ctx context.Context
	}

	InitParams struct {
		// Working directory in the container.
		Dir string
		// Initial process environment.
		Env []string
		// Absolute path of initial process in the container. Overrides name.
		Path string
		// Initial process argv.
		Args []string

		// Mapped Uid in user namespace.
		Uid int
		// Mapped Gid in user namespace.
		Gid int
		// Hostname value in UTS namespace.
		Hostname string
		// Sequential container setup ops.
		*Ops
		// Extra seccomp options.
		Seccomp seccomp.SyscallOpts

		Flags HardeningFlags
	}

	Ops []Op
	Op  interface {
		apply(params *InitParams) error

		Is(op Op) bool
		fmt.Stringer
	}
)

func (p *Container) Start() error {
	if p.cmd != nil {
		return errors.New("sandbox: already started")
	}
	if p.Ops == nil || len(*p.Ops) == 0 {
		return errors.New("sandbox: starting an empty container")
	}

	ctx, cancel := context.WithCancel(p.ctx)
	p.cancel = cancel

	var cloneFlags uintptr = syscall.CLONE_NEWIPC |
		syscall.CLONE_NEWUTS |
		syscall.CLONE_NEWCGROUP
	if p.Flags&FAllowNet == 0 {
		cloneFlags |= syscall.CLONE_NEWNET
	}

	// map to overflow id to work around ownership checks
	if p.Uid < 1 {
		p.Uid = OverflowUid()
	}
	if p.Gid < 1 {
		p.Gid = OverflowGid()
	}

	if p.CommandContext != nil {
		p.cmd = p.CommandContext(ctx)
	} else {
		p.cmd = exec.CommandContext(ctx, MustExecutable())
		p.cmd.Args = []string{"init"}
	}

	p.cmd.Stdin, p.cmd.Stdout, p.cmd.Stderr = p.Stdin, p.Stdout, p.Stderr
	p.cmd.Cancel, p.cmd.WaitDelay = p.Cancel, p.WaitDelay
	p.cmd.Dir = "/"
	p.cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid:    p.Flags&FAllowTTY == 0,
		Pdeathsig: syscall.SIGKILL,

		Cloneflags: cloneFlags |
			syscall.CLONE_NEWUSER |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS,

		// remain privileged for setup
		AmbientCaps: []uintptr{CAP_SYS_ADMIN},

		UseCgroupFD: p.Cgroup != nil,
	}
	if p.cmd.SysProcAttr.UseCgroupFD {
		p.cmd.SysProcAttr.CgroupFD = *p.Cgroup
	}

	// place setup pipe before user supplied extra files, this is later restored by init
	if fd, e, err := proc.Setup(&p.cmd.ExtraFiles); err != nil {
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

func (p *Container) Serve() error {
	if p.setup == nil {
		panic("invalid serve")
	}

	if p.Path != "" && !path.IsAbs(p.Path) {
		return msg.WrapErr(syscall.EINVAL,
			fmt.Sprintf("invalid executable path %q", p.Path))
	}

	if p.Path == "" {
		if p.name == "" {
			p.Path = os.Getenv("SHELL")
			if !path.IsAbs(p.Path) {
				return msg.WrapErr(syscall.EBADE,
					"no command specified and $SHELL is invalid")
			}
			p.name = path.Base(p.Path)
		} else if path.IsAbs(p.name) {
			p.Path = p.name
		} else if v, err := exec.LookPath(p.name); err != nil {
			return msg.WrapErr(err, err.Error())
		} else {
			p.Path = v
		}
	}

	setup := p.setup
	p.setup = nil
	return setup.Encode(
		&initParams{
			p.InitParams,
			syscall.Getuid(),
			syscall.Getgid(),
			len(p.ExtraFiles),
			msg.IsVerbose(),
		},
	)
}

func (p *Container) Wait() error { defer p.cancel(); return p.cmd.Wait() }

func (p *Container) String() string {
	return fmt.Sprintf("argv: %q, flags: %#x, seccomp: %#x",
		p.Args, p.Flags, int(p.Flags.seccomp(p.Seccomp)))
}

func New(ctx context.Context, name string, args ...string) *Container {
	return &Container{name: name, ctx: ctx,
		InitParams: InitParams{Args: append([]string{name}, args...), Dir: "/", Ops: new(Ops)},
	}
}
