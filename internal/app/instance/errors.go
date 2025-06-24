package instance

import (
	"syscall"

	"git.gensokyo.uk/security/hakurei/internal/app"
	"git.gensokyo.uk/security/hakurei/internal/app/internal/setuid"
)

func PrintRunStateErr(whence int, rs *app.RunState, runErr error) (code int) {
	switch whence {
	case ISetuid:
		return setuid.PrintRunStateErr(rs, runErr)
	default:
		panic(syscall.EINVAL)
	}
}
