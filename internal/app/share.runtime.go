package app

import (
	"path"

	"git.ophivana.moe/security/fortify/acl"
	"git.ophivana.moe/security/fortify/internal/system"
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
	seal.sys.bwrap.SetEnv[xdgRuntimeDir] = seal.sys.runtime
	seal.sys.bwrap.SetEnv[xdgSessionClass] = "user"
	seal.sys.bwrap.SetEnv[xdgSessionType] = "tty"

	// ensure RunDir (e.g. `/run/user/%d/fortify`)
	seal.sys.Ensure(seal.RunDirPath, 0700)
	seal.sys.UpdatePermType(system.User, seal.RunDirPath, acl.Execute)

	// ensure runtime directory ACL (e.g. `/run/user/%d`)
	seal.sys.Ensure(seal.RuntimePath, 0700) // ensure this dir in case XDG_RUNTIME_DIR is unset
	seal.sys.UpdatePermType(system.User, seal.RuntimePath, acl.Execute)

	// ensure process-specific share local to XDG_RUNTIME_DIR (e.g. `/run/user/%d/fortify/%s`)
	seal.shareLocal = path.Join(seal.RunDirPath, seal.id)
	seal.sys.Ephemeral(system.Process, seal.shareLocal, 0700)
	seal.sys.UpdatePerm(seal.shareLocal, acl.Execute)
}
