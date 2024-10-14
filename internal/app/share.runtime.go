package app

import (
	"path"

	"git.ophivana.moe/cat/fortify/acl"
	"git.ophivana.moe/cat/fortify/internal/state"
)

const (
	xdgRuntimeDir   = "XDG_RUNTIME_DIR"
	xdgSessionClass = "XDG_SESSION_CLASS"
	xdgSessionType  = "XDG_SESSION_TYPE"
)

// shareRuntime queues actions for sharing/ensuring the runtime and share directories
func (seal *appSeal) shareRuntime() {
	// mount tmpfs on inner runtime (e.g. `/run/user/%d`)
	seal.sys.bwrap.Tmpfs("/run/user", 1*1024*1024)
	seal.sys.bwrap.Tmpfs(seal.sys.runtime, 8*1024*1024)

	// point to inner runtime path `/run/user/%d`
	seal.sys.setEnv(xdgRuntimeDir, seal.sys.runtime)
	seal.sys.setEnv(xdgSessionClass, "user")
	seal.sys.setEnv(xdgSessionType, "tty")

	// ensure RunDir (e.g. `/run/user/%d/fortify`)
	seal.sys.ensure(seal.RunDirPath, 0700)
	seal.sys.updatePermTag(state.EnableLength, seal.RunDirPath, acl.Execute)

	// ensure runtime directory ACL (e.g. `/run/user/%d`)
	seal.sys.updatePermTag(state.EnableLength, seal.RuntimePath, acl.Execute)

	// ensure Share (e.g. `/tmp/fortify.%d`)
	// acl is unnecessary as this directory is world executable
	seal.sys.ensure(seal.SharePath, 0701)

	// ensure process-specific share (e.g. `/tmp/fortify.%d/%s`)
	// acl is unnecessary as this directory is world executable
	seal.share = path.Join(seal.SharePath, seal.id.String())
	seal.sys.ensureEphemeral(seal.share, 0701)

	// ensure process-specific share local to XDG_RUNTIME_DIR (e.g. `/run/user/%d/fortify/%s`)
	seal.shareLocal = path.Join(seal.RunDirPath, seal.id.String())
	seal.sys.ensureEphemeral(seal.shareLocal, 0700)
	seal.sys.updatePerm(seal.shareLocal, acl.Execute)
}
