package app

import (
	"encoding/gob"

	"hakurei.app/container/check"
	"hakurei.app/container/comp"
	"hakurei.app/container/fhs"
	"hakurei.app/hst"
	"hakurei.app/system"
	"hakurei.app/system/acl"
)

func init() { gob.Register(spTmpdirOp{}) }

// spTmpdirOp sets up TMPDIR inside the container.
type spTmpdirOp struct{}

func (s spTmpdirOp) toSystem(state *outcomeStateSys) error {
	if state.Container.Flags&hst.FShareTmpdir != 0 {
		tmpdir, tmpdirInst := s.commonPaths(state.outcomeState)
		state.sys.Ensure(tmpdir, 0700)
		state.sys.UpdatePermType(system.User, tmpdir, acl.Execute)
		state.sys.Ensure(tmpdirInst, 01700)
		state.sys.UpdatePermType(system.User, tmpdirInst, acl.Read, acl.Write, acl.Execute)
	}
	return nil
}

func (s spTmpdirOp) toContainer(state *outcomeStateParams) error {
	if state.Container.Flags&hst.FShareTmpdir != 0 {
		_, tmpdirInst := s.commonPaths(state.outcomeState)
		state.params.Bind(tmpdirInst, fhs.AbsTmp, comp.BindWritable)
	} else {
		state.params.Tmpfs(fhs.AbsTmp, 0, 01777)
	}
	return nil
}

func (s spTmpdirOp) commonPaths(state *outcomeState) (tmpdir, tmpdirInst *check.Absolute) {
	tmpdir = state.sc.SharePath.Append("tmpdir")
	tmpdirInst = tmpdir.Append(state.identity.String())
	return
}
