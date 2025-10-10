package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"
	"sync/atomic"

	"hakurei.app/hst"
	"hakurei.app/internal/app/state"
	"hakurei.app/message"
	"hakurei.app/system"
)

func newWithMessage(msg string) error { return newWithMessageError(msg, os.ErrInvalid) }
func newWithMessageError(msg string, err error) error {
	return &hst.AppError{Step: "finalise", Err: err, Msg: msg}
}

// An outcome is the runnable state of a hakurei container via [hst.Config].
type outcome struct {
	// Supplementary group ids. Populated during finalise.
	supp []string
	// Resolved priv side operating system interactions. Populated during finalise.
	sys *system.I
	// Transmitted to shim. Populated during finalise.
	state *outcomeState
	// Kept for saving to [state].
	config *hst.Config

	// Whether the current process is in outcome.main.
	active atomic.Bool

	ctx context.Context
	syscallDispatcher
}

func (k *outcome) finalise(ctx context.Context, msg message.Msg, id *state.ID, config *hst.Config) error {
	if ctx == nil || id == nil {
		// unreachable
		panic("invalid call to finalise")
	}
	if k.ctx != nil || k.sys != nil || k.state != nil {
		// unreachable
		panic("attempting to finalise twice")
	}
	k.ctx = ctx

	if err := config.Validate(); err != nil {
		return err
	}

	// hsu expects numerical group ids
	supp := make([]string, len(config.Groups))
	for i, name := range config.Groups {
		if gid, err := k.lookupGroupId(name); err != nil {
			var unknownGroupError user.UnknownGroupError
			if errors.As(err, &unknownGroupError) {
				return newWithMessageError(fmt.Sprintf("unknown group %q", name), unknownGroupError)
			} else {
				return &hst.AppError{Step: "look up group by name", Err: err}
			}
		} else {
			supp[i] = gid
		}
	}

	// early validation complete at this point
	s := newOutcomeState(k.syscallDispatcher, msg, id, config, &Hsu{k: k})
	if err := s.populateLocal(k.syscallDispatcher, msg); err != nil {
		return err
	}

	sys := system.New(k.ctx, msg, s.uid.unwrap())
	if err := s.newSys(config, sys).toSystem(); err != nil {
		return err
	}

	k.sys = sys
	k.supp = supp
	k.state = s
	k.config = config
	return nil
}
