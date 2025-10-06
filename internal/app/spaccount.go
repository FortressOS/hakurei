package app

import (
	"fmt"
	"syscall"

	"hakurei.app/container"
	"hakurei.app/hst"
)

// spAccountOp sets up user account emulation inside the container.
type spAccountOp struct{}

func (s spAccountOp) toSystem(state *outcomeStateSys, _ *hst.Config) error {
	const fallbackUsername = "chronos"

	// do checks here to fail before fork/exec
	if state.Container == nil || state.Container.Home == nil || state.Container.Shell == nil {
		// unreachable
		return syscall.ENOTRECOVERABLE
	}
	if state.Container.Username == "" {
		state.Container.Username = fallbackUsername
	} else if !isValidUsername(state.Container.Username) {
		return newWithMessage(fmt.Sprintf("invalid user name %q", state.Container.Username))
	}
	return nil
}

func (s spAccountOp) toContainer(state *outcomeStateParams) error {
	state.params.Dir = state.Container.Home
	state.env["HOME"] = state.Container.Home.String()
	state.env["USER"] = state.Container.Username
	state.env["SHELL"] = state.Container.Shell.String()

	state.params.
		Place(container.AbsFHSEtc.Append("passwd"),
			[]byte(state.Container.Username+":x:"+
				state.mapuid.String()+":"+
				state.mapgid.String()+
				":Hakurei:"+
				state.Container.Home.String()+":"+
				state.Container.Shell.String()+"\n")).
		Place(container.AbsFHSEtc.Append("group"),
			[]byte("hakurei:x:"+state.mapgid.String()+":\n"))

	return nil
}
