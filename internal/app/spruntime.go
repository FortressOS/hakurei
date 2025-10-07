package app

import (
	"hakurei.app/container"
	"hakurei.app/container/check"
	"hakurei.app/container/fhs"
	"hakurei.app/hst"
	"hakurei.app/system"
	"hakurei.app/system/acl"
)

// spRuntimeOp sets up XDG_RUNTIME_DIR inside the container.
type spRuntimeOp struct{}

func (s spRuntimeOp) toSystem(state *outcomeStateSys, _ *hst.Config) error {
	runtimeDir, runtimeDirInst := s.commonPaths(state.outcomeState)
	state.sys.Ensure(runtimeDir, 0700)
	state.sys.UpdatePermType(system.User, runtimeDir, acl.Execute)
	state.sys.Ensure(runtimeDirInst, 0700)
	state.sys.UpdatePermType(system.User, runtimeDirInst, acl.Read, acl.Write, acl.Execute)
	return nil
}

func (s spRuntimeOp) toContainer(state *outcomeStateParams) error {
	const (
		xdgRuntimeDir   = "XDG_RUNTIME_DIR"
		xdgSessionClass = "XDG_SESSION_CLASS"
		xdgSessionType  = "XDG_SESSION_TYPE"
	)

	state.runtimeDir = fhs.AbsRunUser.Append(state.mapuid.String())
	state.env[xdgRuntimeDir] = state.runtimeDir.String()
	state.env[xdgSessionClass] = "user"
	state.env[xdgSessionType] = "tty"

	_, runtimeDirInst := s.commonPaths(state.outcomeState)
	state.params.Tmpfs(fhs.AbsRunUser, 1<<12, 0755)
	state.params.Bind(runtimeDirInst, state.runtimeDir, container.BindWritable)
	return nil
}

func (s spRuntimeOp) commonPaths(state *outcomeState) (runtimeDir, runtimeDirInst *check.Absolute) {
	runtimeDir = state.sc.SharePath.Append("runtime")
	runtimeDirInst = runtimeDir.Append(state.identity.String())
	return
}
