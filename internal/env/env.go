// Package env provides the [Paths] struct for efficiently building paths from the environment.
package env

import (
	"log"
	"os"
	"strconv"

	"hakurei.app/container/check"
	"hakurei.app/hst"
)

// Paths holds paths copied from the environment and is used to create [hst.Paths].
type Paths struct {
	// TempDir is returned by [os.TempDir].
	TempDir *check.Absolute
	// RuntimePath is copied from $XDG_RUNTIME_DIR.
	RuntimePath *check.Absolute
}

// Copy expands [Paths] into [hst.Paths].
func (env *Paths) Copy(v *hst.Paths, userid int) {
	if env == nil || env.TempDir == nil || v == nil {
		panic("attempting to use an invalid Paths")
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

// CopyPaths returns a populated [Paths].
func CopyPaths() *Paths { return CopyPathsFunc(log.Fatalf, os.TempDir, os.Getenv) }

// CopyPathsFunc returns a populated [Paths],
// using the provided [log.Fatalf], [os.TempDir], [os.Getenv] functions.
func CopyPathsFunc(
	fatalf func(format string, v ...any),
	tempdir func() string,
	getenv func(key string) string,
) *Paths {
	const xdgRuntimeDir = "XDG_RUNTIME_DIR"

	var env Paths

	if tempDir, err := check.NewAbs(tempdir()); err != nil {
		fatalf("invalid TMPDIR: %v", err)
		panic("unreachable")
	} else {
		env.TempDir = tempDir
	}

	if a, err := check.NewAbs(getenv(xdgRuntimeDir)); err == nil {
		env.RuntimePath = a
	}

	return &env
}
