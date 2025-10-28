package app

import (
	"encoding/gob"
	"fmt"
	"syscall"

	"hakurei.app/container/fhs"
	"hakurei.app/internal/validate"
)

func init() { gob.Register(spAccountOp{}) }

// spAccountOp sets up user account emulation inside the container.
type spAccountOp struct{}

func (s spAccountOp) toSystem(state *outcomeStateSys) error {
	// do checks here to fail before fork/exec
	if state.Container == nil || state.Container.Home == nil || state.Container.Shell == nil {
		// unreachable
		return syscall.ENOTRECOVERABLE
	}

	// default is applied in toContainer
	if state.Container.Username != "" && !validate.IsValidUsername(state.Container.Username) {
		return newWithMessage(fmt.Sprintf("invalid user name %q", state.Container.Username))
	}
	return nil
}

func (s spAccountOp) toContainer(state *outcomeStateParams) error {
	const fallbackUsername = "chronos"

	username := state.Container.Username
	if username == "" {
		username = fallbackUsername
	}

	state.params.Dir = state.Container.Home
	state.env["HOME"] = state.Container.Home.String()
	state.env["USER"] = username
	state.env["SHELL"] = state.Container.Shell.String()

	state.params.
		Place(fhs.AbsEtc.Append("passwd"),
			[]byte(username+":x:"+
				state.mapuid.String()+":"+
				state.mapgid.String()+
				":Hakurei:"+
				state.Container.Home.String()+":"+
				state.Container.Shell.String()+"\n")).
		Place(fhs.AbsEtc.Append("group"),
			[]byte("hakurei:x:"+state.mapgid.String()+":\n"))

	return nil
}
