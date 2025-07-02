package internal

import (
	"git.gensokyo.uk/security/hakurei/internal/hlog"
	"git.gensokyo.uk/security/hakurei/sandbox"
	"git.gensokyo.uk/security/hakurei/system"
)

func InstallOutput(verbose bool) {
	hlog.Store(verbose)
	sandbox.SetOutput(hlog.Output{})
	system.SetOutput(hlog.Output{})
}
