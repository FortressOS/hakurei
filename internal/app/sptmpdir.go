package app

import (
	"hakurei.app/container"
	"hakurei.app/container/check"
	"hakurei.app/hst"
	"hakurei.app/system"
	"hakurei.app/system/acl"
)

// spTmpdirOp sets up TMPDIR inside the container.
type spTmpdirOp struct{}

func (s spTmpdirOp) toSystem(state *outcomeStateSys, _ *hst.Config) error {
	tmpdir, tmpdirInst := s.commonPaths(state.outcomeState)
	state.sys.Ensure(tmpdir, 0700)
	state.sys.UpdatePermType(system.User, tmpdir, acl.Execute)
	state.sys.Ensure(tmpdirInst, 01700)
	state.sys.UpdatePermType(system.User, tmpdirInst, acl.Read, acl.Write, acl.Execute)
	return nil
}

func (s spTmpdirOp) toContainer(state *outcomeStateParams) error {
	// mount inner /tmp from share so it shares persistence and storage behaviour of host /tmp
	_, tmpdirInst := s.commonPaths(state.outcomeState)
	state.params.Bind(tmpdirInst, container.AbsFHSTmp, container.BindWritable)
	return nil
}

func (s spTmpdirOp) commonPaths(state *outcomeState) (tmpdir, tmpdirInst *check.Absolute) {
	tmpdir = state.sc.SharePath.Append("tmpdir")
	tmpdirInst = tmpdir.Append(state.identity.String())
	return
}
