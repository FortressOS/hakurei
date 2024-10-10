package app

import (
	"os"
	"path"

	"git.ophivana.moe/cat/fortify/acl"
	"git.ophivana.moe/cat/fortify/helper/bwrap"
	"git.ophivana.moe/cat/fortify/internal/state"
)

const (
	xdgRuntimeDir   = "XDG_RUNTIME_DIR"
	xdgSessionClass = "XDG_SESSION_CLASS"
	xdgSessionType  = "XDG_SESSION_TYPE"

	shell = "SHELL"
)

// shareRuntime queues actions for sharing/ensuring the runtime and share directories
func (seal *appSeal) shareRuntime() {
	// look up shell
	if s, ok := os.LookupEnv(shell); ok {
		seal.sys.setEnv(shell, s)
	}

	// mount tmpfs on inner runtime (e.g. `/run/user/%d`)
	seal.sys.bwrap.Tmpfs = append(seal.sys.bwrap.Tmpfs,
		bwrap.PermConfig[bwrap.TmpfsConfig]{
			Path: bwrap.TmpfsConfig{
				Size: 1 * 1024 * 1024,
				Dir:  "/run/user",
			},
		},
		bwrap.PermConfig[bwrap.TmpfsConfig]{
			Path: bwrap.TmpfsConfig{
				Size: 8 * 1024 * 1024,
				Dir:  seal.sys.runtime,
			},
		},
	)

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

func (seal *appSeal) shareRuntimeChild() string {
	// ensure child runtime parent directory (e.g. `/tmp/fortify.%d/runtime`)
	targetRuntimeParent := path.Join(seal.SharePath, "runtime")
	seal.sys.ensure(targetRuntimeParent, 0700)
	seal.sys.updatePermTag(state.EnableLength, targetRuntimeParent, acl.Execute)

	// ensure child runtime directory (e.g. `/tmp/fortify.%d/runtime/%d`)
	targetRuntime := path.Join(targetRuntimeParent, seal.sys.Uid)
	seal.sys.ensure(targetRuntime, 0700)
	seal.sys.updatePermTag(state.EnableLength, targetRuntime, acl.Read, acl.Write, acl.Execute)

	// point to ensured runtime path
	seal.sys.setEnv(xdgRuntimeDir, targetRuntime)
	seal.sys.setEnv(xdgSessionClass, "user")
	seal.sys.setEnv(xdgSessionType, "tty")

	return targetRuntime
}
