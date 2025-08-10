package hst

import "hakurei.app/container"

// Paths contains environment-dependent paths used by hakurei.
type Paths struct {
	// temporary directory returned by [os.TempDir] (usually `/tmp`)
	TempDir *container.Absolute `json:"temp_dir"`
	// path to shared directory (usually `/tmp/hakurei.%d`)
	SharePath *container.Absolute `json:"share_path"`
	// XDG_RUNTIME_DIR value (usually `/run/user/%d`)
	RuntimePath *container.Absolute `json:"runtime_path"`
	// application runtime directory (usually `/run/user/%d/hakurei`)
	RunDirPath *container.Absolute `json:"run_dir_path"`
}
