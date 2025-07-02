package internal

import (
	"hakurei.app/container"
	"hakurei.app/internal/hlog"
	"hakurei.app/system"
)

func InstallOutput(verbose bool) {
	hlog.Store(verbose)
	container.SetOutput(hlog.Output{})
	system.SetOutput(hlog.Output{})
}
