package system

import (
	"git.gensokyo.uk/security/hakurei/container"
)

var msg container.Msg = new(container.DefaultMsg)

func SetOutput(v container.Msg) {
	if v == nil {
		msg = new(container.DefaultMsg)
	} else {
		msg = v
	}
}

func wrapErrSuffix(err error, a ...any) error {
	if err == nil {
		return nil
	}
	return msg.WrapErr(err, append(a, err)...)
}
