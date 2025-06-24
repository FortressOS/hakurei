package system

import "git.gensokyo.uk/security/hakurei/sandbox"

var msg sandbox.Msg = new(sandbox.DefaultMsg)

func SetOutput(v sandbox.Msg) {
	if v == nil {
		msg = new(sandbox.DefaultMsg)
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
