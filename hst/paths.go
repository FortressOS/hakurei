package hst

// Paths contains environment-dependent paths used by hakurei.
type Paths struct {
	// path to shared directory (usually `/tmp/hakurei.%d`)
	SharePath string `json:"share_path"`
	// XDG_RUNTIME_DIR value (usually `/run/user/%d`)
	RuntimePath string `json:"runtime_path"`
	// application runtime directory (usually `/run/user/%d/hakurei`)
	RunDirPath string `json:"run_dir_path"`
}
