// Package app implements high-level hakurei container behaviour.
package app

import (
	"syscall"
	"time"
)

type SealedApp interface {
	// Run commits sealed system setup and starts the app process.
	Run(rs *RunState) error
}

// RunState stores the outcome of a call to [SealedApp.Run].
type RunState struct {
	// Time is the exact point in time where the process was created.
	// Location must be set to UTC.
	//
	// Time is nil if no process was ever created.
	Time *time.Time
	// RevertErr is stored by the deferred revert call.
	RevertErr error
	// WaitErr is the generic error value created by the standard library.
	WaitErr error

	syscall.WaitStatus
}

// SetStart stores the current time in [RunState] once.
func (rs *RunState) SetStart() {
	if rs.Time != nil {
		panic("attempted to store time twice")
	}
	now := time.Now().UTC()
	rs.Time = &now
}
