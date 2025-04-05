// Package fst exports shared fortify types.
package fst

import (
	"syscall"
	"time"
)

type App interface {
	// ID returns a copy of [fst.ID] held by App.
	ID() ID

	// Seal determines the outcome of config as a [SealedApp].
	// The value of config might be overwritten and must not be used again.
	Seal(config *Config) (SealedApp, error)

	String() string
}

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

// Paths contains environment-dependent paths used by fortify.
type Paths struct {
	// path to shared directory (usually `/tmp/fortify.%d`)
	SharePath string `json:"share_path"`
	// XDG_RUNTIME_DIR value (usually `/run/user/%d`)
	RuntimePath string `json:"runtime_path"`
	// application runtime directory (usually `/run/user/%d/fortify`)
	RunDirPath string `json:"run_dir_path"`
}
