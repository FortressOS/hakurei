package app

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"os/user"
	"slices"
	"strings"
	"sync/atomic"
	"syscall"

	"hakurei.app/container"
	"hakurei.app/hst"
	"hakurei.app/internal/app/state"
	"hakurei.app/system"
	"hakurei.app/system/acl"
)

func newWithMessage(msg string) error { return newWithMessageError(msg, os.ErrInvalid) }
func newWithMessageError(msg string, err error) error {
	return &hst.AppError{Step: "finalise", Err: err, Msg: msg}
}

// An outcome is the runnable state of a hakurei container via [hst.Config].
type outcome struct {
	// initial [hst.Config] gob stream for state data;
	// this is prepared ahead of time as config is clobbered during seal creation
	ct io.WriterTo

	sys *system.I
	ctx context.Context

	container container.Params

	// TODO(ophestra): move this to the system op
	sync *os.File

	// Populated during outcome.finalise.
	proc *finaliseProcess

	// Whether the current process is in outcome.main.
	active atomic.Bool

	syscallDispatcher
}

func (k *outcome) finalise(ctx context.Context, msg container.Msg, id *state.ID, config *hst.Config) error {
	const (
		// only used for a nil configured env map
		envAllocSize = 1 << 6
	)

	var kp finaliseProcess

	if ctx == nil || id == nil {
		// unreachable
		panic("invalid call to finalise")
	}
	if k.ctx != nil || k.proc != nil {
		// unreachable
		panic("attempting to finalise twice")
	}
	k.ctx = ctx

	if config == nil {
		return newWithMessage("invalid configuration")
	}
	if config.Home == nil {
		return newWithMessage("invalid path to home directory")
	}

	// TODO(ophestra): do not clobber during finalise
	{
		// encode initial configuration for state tracking
		ct := new(bytes.Buffer)
		if err := gob.NewEncoder(ct).Encode(config); err != nil {
			return &hst.AppError{Step: "encode initial config", Err: err}
		}
		k.ct = ct
	}

	// allowed identity range 0 to 9999, this is checked again in hsu
	if config.Identity < 0 || config.Identity > 9999 {
		return newWithMessage(fmt.Sprintf("identity %d out of range", config.Identity))
	}

	kp.supp = make([]string, len(config.Groups))
	for i, name := range config.Groups {
		if gid, err := k.lookupGroupId(name); err != nil {
			var unknownGroupError user.UnknownGroupError
			if errors.As(err, &unknownGroupError) {
				return newWithMessageError(fmt.Sprintf("unknown group %q", name), unknownGroupError)
			} else {
				return &hst.AppError{Step: "look up group by name", Err: err}
			}
		} else {
			kp.supp[i] = gid
		}
	}

	// permissive defaults
	if config.Container == nil {
		msg.Verbose("container configuration not supplied, PROCEED WITH CAUTION")

		if config.Shell == nil {
			config.Shell = container.AbsFHSRoot.Append("bin", "sh")
			s, _ := k.lookupEnv("SHELL")
			if a, err := container.NewAbs(s); err == nil {
				config.Shell = a
			}
		}

		// hsu clears the environment so resolve paths early
		if config.Path == nil {
			if len(config.Args) > 0 {
				if p, err := k.lookPath(config.Args[0]); err != nil {
					return &hst.AppError{Step: "look up executable file", Err: err}
				} else if config.Path, err = container.NewAbs(p); err != nil {
					return newWithMessageError(err.Error(), err)
				}
			} else {
				config.Path = config.Shell
			}
		}

		conf := &hst.ContainerConfig{
			Userns:       true,
			HostNet:      true,
			HostAbstract: true,
			Tty:          true,

			Filesystem: []hst.FilesystemConfigJSON{
				// autoroot, includes the home directory
				{FilesystemConfig: &hst.FSBind{
					Target:  container.AbsFHSRoot,
					Source:  container.AbsFHSRoot,
					Write:   true,
					Special: true,
				}},
			},
		}

		// bind GPU stuff
		if config.Enablements.Unwrap()&(hst.EX11|hst.EWayland) != 0 {
			conf.Filesystem = append(conf.Filesystem, hst.FilesystemConfigJSON{FilesystemConfig: &hst.FSBind{Source: container.AbsFHSDev.Append("dri"), Device: true, Optional: true}})
		}
		// opportunistically bind kvm
		conf.Filesystem = append(conf.Filesystem, hst.FilesystemConfigJSON{FilesystemConfig: &hst.FSBind{Source: container.AbsFHSDev.Append("kvm"), Device: true, Optional: true}})

		// hide nscd from container if present
		nscd := container.AbsFHSVar.Append("run/nscd")
		if _, err := k.stat(nscd.String()); !errors.Is(err, fs.ErrNotExist) {
			conf.Filesystem = append(conf.Filesystem, hst.FilesystemConfigJSON{FilesystemConfig: &hst.FSEphemeral{Target: nscd}})
		}

		// do autoetc last
		conf.Filesystem = append(conf.Filesystem,
			hst.FilesystemConfigJSON{FilesystemConfig: &hst.FSBind{
				Target:  container.AbsFHSEtc,
				Source:  container.AbsFHSEtc,
				Special: true,
			}},
		)

		config.Container = conf
	}

	// late nil checks for pd behaviour
	if config.Shell == nil {
		return newWithMessage("invalid shell path")
	}
	if config.Path == nil {
		return newWithMessage("invalid program path")
	}

	// enforce bounds and default early
	kp.waitDelay = shimWaitTimeout
	if config.Container.WaitDelay <= 0 {
		kp.waitDelay += DefaultShimWaitDelay
	} else if config.Container.WaitDelay > MaxShimWaitDelay {
		kp.waitDelay += MaxShimWaitDelay
	} else {
		kp.waitDelay += config.Container.WaitDelay
	}

	s := outcomeState{
		ID:       id,
		Identity: config.Identity,
		UserID:   (&Hsu{k: k}).MustIDMsg(msg),
		EnvPaths: copyPaths(k.syscallDispatcher),

		// TODO(ophestra): apply pd behaviour here instead of clobbering hst.Config
		Container: config.Container,
	}
	if s.Container.MapRealUID {
		s.Mapuid, s.Mapgid = k.getuid(), k.getgid()
	} else {
		s.Mapuid, s.Mapgid = k.overflowUid(msg), k.overflowGid(msg)
	}

	// TODO(ophestra): duplicate in shim (params to shim)
	if err := s.populateLocal(k.syscallDispatcher, msg); err != nil {
		return err
	}
	kp.runDirPath, kp.identity, kp.id = s.sc.RunDirPath, s.identity, s.id
	k.sys = system.New(k.ctx, msg, s.uid.unwrap())

	{
		ops := []outcomeOp{
			// must run first
			&spParamsOp{Path: config.Path, Args: config.Args},

			// TODO(ophestra): move this late for #8 and #9
			spFilesystemOp{},

			spRuntimeOp{},
			spTmpdirOp{},
			&spAccountOp{Home: config.Home, Username: config.Username, Shell: config.Shell},
		}

		et := config.Enablements.Unwrap()
		if et&hst.EWayland != 0 {
			ops = append(ops, &spWaylandOp{sync: &k.sync})
		}
		if et&hst.EX11 != 0 {
			ops = append(ops, &spX11Op{})
		}
		if et&hst.EPulse != 0 {
			ops = append(ops, &spPulseOp{})
		}
		if et&hst.EDBus != 0 {
			ops = append(ops, &spDBusOp{})
		}

		stateSys := outcomeStateSys{sys: k.sys, outcomeState: &s}
		for _, op := range ops {
			if err := op.toSystem(&stateSys, config); err != nil {
				return err
			}
		}

		// TODO(ophestra): move to shim
		stateParams := outcomeStateParams{params: &k.container, outcomeState: &s}
		if s.Container.Env == nil {
			stateParams.env = make(map[string]string, envAllocSize)
		} else {
			stateParams.env = maps.Clone(s.Container.Env)
		}
		for _, op := range ops {
			if err := op.toContainer(&stateParams); err != nil {
				return err
			}
		}
		// flatten and sort env for deterministic behaviour
		k.container.Env = make([]string, 0, len(stateParams.env))
		for key, value := range stateParams.env {
			if strings.IndexByte(key, '=') != -1 {
				return &hst.AppError{Step: "flatten environment", Err: syscall.EINVAL,
					Msg: fmt.Sprintf("invalid environment variable %s", key)}
			}
			k.container.Env = append(k.container.Env, key+"="+value)
		}
		slices.Sort(k.container.Env)
	}

	// mount root read-only as the final setup Op
	// TODO(ophestra): move this to spFilesystemOp after #8 and #9
	k.container.Remount(container.AbsFHSRoot, syscall.MS_RDONLY)

	// append ExtraPerms last
	for _, p := range config.ExtraPerms {
		if p == nil || p.Path == nil {
			continue
		}

		if p.Ensure {
			k.sys.Ensure(p.Path, 0700)
		}

		perms := make(acl.Perms, 0, 3)
		if p.Read {
			perms = append(perms, acl.Read)
		}
		if p.Write {
			perms = append(perms, acl.Write)
		}
		if p.Execute {
			perms = append(perms, acl.Execute)
		}
		k.sys.UpdatePermType(system.User, p.Path, perms...)
	}

	k.proc = &kp
	return nil
}
