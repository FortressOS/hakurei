package app

import (
	"fmt"
	"syscall"

	"hakurei.app/container"
	"hakurei.app/hst"
)

// spAccountOp sets up user account emulation inside the container.
type spAccountOp struct {
	// Inner directory to use as the home directory of the emulated user.
	Home *container.Absolute
	// String matching the default NAME_REGEX value from adduser to use as the username of the emulated user.
	Username string
	// Pathname of shell to use for the emulated user.
	Shell *container.Absolute
}

func (s *spAccountOp) toSystem(*outcomeStateSys, *hst.Config) error {
	const fallbackUsername = "chronos"

	// do checks here to fail before fork/exec
	if s.Home == nil || s.Shell == nil {
		// unreachable
		return syscall.ENOTRECOVERABLE
	}
	if s.Username == "" {
		s.Username = fallbackUsername
	} else if !isValidUsername(s.Username) {
		return newWithMessage(fmt.Sprintf("invalid user name %q", s.Username))
	}
	return nil
}

func (s *spAccountOp) toContainer(state *outcomeStateParams) error {
	state.params.Dir = s.Home
	state.env["HOME"] = s.Home.String()
	state.env["USER"] = s.Username
	state.env["SHELL"] = s.Shell.String()

	state.params.
		Place(container.AbsFHSEtc.Append("passwd"),
			[]byte(s.Username+":x:"+
				state.mapuid.String()+":"+
				state.mapgid.String()+
				":Hakurei:"+
				s.Home.String()+":"+
				s.Shell.String()+"\n")).
		Place(container.AbsFHSEtc.Append("group"),
			[]byte("hakurei:x:"+state.mapgid.String()+":\n"))

	return nil
}
