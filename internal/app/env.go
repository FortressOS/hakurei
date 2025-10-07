package app

import (
	"strconv"

	"hakurei.app/container/check"
	"hakurei.app/hst"
)

// EnvPaths holds paths copied from the environment and is used to create [hst.Paths].
type EnvPaths struct {
	// TempDir is returned by [os.TempDir].
	TempDir *check.Absolute
	// RuntimePath is copied from $XDG_RUNTIME_DIR.
	RuntimePath *check.Absolute
}

// Copy expands [EnvPaths] into [hst.Paths].
func (env *EnvPaths) Copy(v *hst.Paths, userid int) {
	if env == nil || env.TempDir == nil || v == nil {
		panic("attempting to use an invalid EnvPaths")
	}

	v.TempDir = env.TempDir
	v.SharePath = env.TempDir.Append("hakurei." + strconv.Itoa(userid))

	if env.RuntimePath == nil {
		// fall back to path in share since hakurei has no hard XDG dependency
		v.RunDirPath = v.SharePath.Append("run")
		v.RuntimePath = v.RunDirPath.Append("compat")
	} else {
		v.RuntimePath = env.RuntimePath
		v.RunDirPath = env.RuntimePath.Append("hakurei")
	}
}

// CopyPaths returns a populated [EnvPaths].
func CopyPaths() *EnvPaths { return copyPaths(direct{}) }

// copyPaths returns a populated [EnvPaths].
func copyPaths(k syscallDispatcher) *EnvPaths {
	const xdgRuntimeDir = "XDG_RUNTIME_DIR"

	var env EnvPaths

	if tempDir, err := check.NewAbs(k.tempdir()); err != nil {
		k.fatalf("invalid TMPDIR: %v", err)
		panic("unreachable")
	} else {
		env.TempDir = tempDir
	}

	r, _ := k.lookupEnv(xdgRuntimeDir)
	if a, err := check.NewAbs(r); err == nil {
		env.RuntimePath = a
	}

	return &env
}
