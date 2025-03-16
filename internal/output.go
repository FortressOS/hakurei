package internal

import (
	"git.gensokyo.uk/security/fortify/internal/fmsg"
	"git.gensokyo.uk/security/fortify/internal/sandbox"
	"git.gensokyo.uk/security/fortify/seccomp"
	"git.gensokyo.uk/security/fortify/system"
)

func InstallFmsg(verbose bool) {
	fmsg.Store(verbose)
	sandbox.SetOutput(fmsg.Output{})
	system.SetOutput(fmsg.Output{})
	if verbose {
		seccomp.SetOutput(fmsg.Verbose)
	}
}
