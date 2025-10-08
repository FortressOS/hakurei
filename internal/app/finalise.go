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
	"sync/atomic"

	"hakurei.app/container"
	"hakurei.app/hst"
	"hakurei.app/internal/app/state"
	"hakurei.app/system"
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
	// only used for a nil configured env map
	const envAllocSize = 1 << 6

	if ctx == nil || id == nil {
		// unreachable
		panic("invalid call to finalise")
	}
	if k.ctx != nil || k.sys != nil || k.proc != nil {
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

	var kp finaliseProcess

	// hsu expects numerical group ids
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
	kp.waitDelay = s.populateEarly(k.syscallDispatcher, msg)

	// TODO(ophestra): duplicate in shim (params to shim)
	if err := s.populateLocal(k.syscallDispatcher, msg); err != nil {
		return err
	}
	kp.runDirPath, kp.identity, kp.id = s.sc.RunDirPath, s.identity, s.id
	sys := system.New(k.ctx, msg, s.uid.unwrap())

	ops := fromConfig(config)

	stateSys := outcomeStateSys{sys: sys, outcomeState: &s}
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

	k.sys = sys
	k.proc = &kp
	return nil
}
