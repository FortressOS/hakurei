package app

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
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

	if err := config.Validate(); err != nil {
		return err
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

	// early validation complete at this point
	s := outcomeState{
		ID:        id,
		Identity:  config.Identity,
		UserID:    (&Hsu{k: k}).MustIDMsg(msg),
		EnvPaths:  copyPaths(k.syscallDispatcher),
		Container: config.Container,
	}

	// enforce bounds and default early
	if s.Container.WaitDelay <= 0 {
		kp.waitDelay = hst.WaitDelayDefault
	} else if s.Container.WaitDelay > hst.WaitDelayMax {
		kp.waitDelay = hst.WaitDelayMax
	} else {
		kp.waitDelay = s.Container.WaitDelay
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
			&spParamsOp{},

			// TODO(ophestra): move this late for #8 and #9
			spFilesystemOp{},

			spRuntimeOp{},
			spTmpdirOp{},
			spAccountOp{},
		}

		et := config.Enablements.Unwrap()
		if et&hst.EWayland != 0 {
			ops = append(ops, &spWaylandOp{})
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
