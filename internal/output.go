package internal

import (
	"git.gensokyo.uk/security/hakurei"
	"git.gensokyo.uk/security/hakurei/internal/hlog"
	"git.gensokyo.uk/security/hakurei/system"
)

func InstallOutput(verbose bool) {
	hlog.Store(verbose)
	hakurei.SetOutput(hlog.Output{})
	system.SetOutput(hlog.Output{})
}
