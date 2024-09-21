package app

import (
	"path"

	"git.ophivana.moe/cat/fortify/acl"
)

const (
	xdgRuntimeDir   = "XDG_RUNTIME_DIR"
	xdgSessionClass = "XDG_SESSION_CLASS"
	xdgSessionType  = "XDG_SESSION_TYPE"
)

// shareRuntime queues actions for sharing/ensuring the runtime and share directories
func (seal *appSeal) shareRuntime() {
	// ensure RunDir (e.g. `/run/user/%d/fortify`)
	seal.sys.ensure(seal.RunDirPath, 0700)

	// ensure runtime directory ACL (e.g. `/run/user/%d`)
	seal.sys.updatePerm(seal.RuntimePath, acl.Execute)

	// ensure Share (e.g. `/tmp/fortify.%d`)
	// acl is unnecessary as this directory is world executable
	seal.sys.ensure(seal.SharePath, 0701)

	// ensure process-specific share (e.g. `/tmp/fortify.%d/%s`)
	// acl is unnecessary as this directory is world executable
	seal.share = path.Join(seal.SharePath, seal.id.String())
	seal.sys.ensureEphemeral(seal.share, 0701)
}

func (seal *appSeal) shareRuntimeChild() string {
	// ensure child runtime parent directory (e.g. `/tmp/fortify.%d/runtime`)
	targetRuntimeParent := path.Join(seal.SharePath, "runtime")
	seal.sys.ensure(targetRuntimeParent, 0700)
	seal.sys.updatePerm(targetRuntimeParent, acl.Execute)

	// ensure child runtime directory (e.g. `/tmp/fortify.%d/runtime/%d`)
	targetRuntime := path.Join(targetRuntimeParent, seal.sys.Uid)
	seal.sys.ensure(targetRuntime, 0700)
	seal.sys.updatePerm(targetRuntime, acl.Read, acl.Write, acl.Execute)

	// point to ensured runtime path
	seal.appendEnv(xdgRuntimeDir, targetRuntime)
	seal.appendEnv(xdgSessionClass, "user")
	seal.appendEnv(xdgSessionType, "tty")

	return targetRuntime
}
