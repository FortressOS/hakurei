package internal

import (
	"git.gensokyo.uk/security/fortify/internal/fmsg"
	"git.gensokyo.uk/security/fortify/seccomp"
)

func InstallFmsg(verbose bool) {
	fmsg.Store(verbose)
	if verbose {
		seccomp.SetOutput(fmsg.Verbose)
	}
}
