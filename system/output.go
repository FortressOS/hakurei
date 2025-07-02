package system

import (
	"git.gensokyo.uk/security/hakurei"
)

var msg hakurei.Msg = new(hakurei.DefaultMsg)

func SetOutput(v hakurei.Msg) {
	if v == nil {
		msg = new(hakurei.DefaultMsg)
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
