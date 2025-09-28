package app

import (
	"strconv"

	"hakurei.app/container"
	"hakurei.app/hst"
)

// CopyPaths populates a [hst.Paths] struct.
func CopyPaths(msg container.Msg, v *hst.Paths, userid int) { copyPaths(direct{}, msg, v, userid) }

// copyPaths populates a [hst.Paths] struct.
func copyPaths(k syscallDispatcher, msg container.Msg, v *hst.Paths, userid int) {
	const xdgRuntimeDir = "XDG_RUNTIME_DIR"

	if tempDir, err := container.NewAbs(k.tempdir()); err != nil {
		k.fatalf("invalid TMPDIR: %v", err)
	} else {
		v.TempDir = tempDir
	}

	v.SharePath = v.TempDir.Append("hakurei." + strconv.Itoa(userid))
	msg.Verbosef("process share directory at %q", v.SharePath)

	r, _ := k.lookupEnv(xdgRuntimeDir)
	if a, err := container.NewAbs(r); err != nil {
		// fall back to path in share since hakurei has no hard XDG dependency
		v.RunDirPath = v.SharePath.Append("run")
		v.RuntimePath = v.RunDirPath.Append("compat")
	} else {
		v.RuntimePath = a
		v.RunDirPath = v.RuntimePath.Append("hakurei")
	}
	msg.Verbosef("runtime directory at %q", v.RunDirPath)
}
