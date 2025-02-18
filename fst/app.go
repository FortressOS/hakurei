package fst

import "context"

type App interface {
	// ID returns a copy of App's unique ID.
	ID() ID
	// Run sets up the system and runs the App.
	Run(ctx context.Context, rs *RunState) error

	Seal(config *Config) error
	String() string
}

// RunState stores the outcome of a call to [App.Run].
type RunState struct {
	// Start is true if fsu is successfully started.
	Start bool
	// ExitCode is the value returned by shim.
	ExitCode int
	// WaitErr is error returned by the underlying wait syscall.
	WaitErr error
}

// Paths contains environment-dependent paths used by fortify.
type Paths struct {
	// path to shared directory e.g. /tmp/fortify.%d
	SharePath string `json:"share_path"`
	// XDG_RUNTIME_DIR value e.g. /run/user/%d
	RuntimePath string `json:"runtime_path"`
	// application runtime directory e.g. /run/user/%d/fortify
	RunDirPath string `json:"run_dir_path"`
}
