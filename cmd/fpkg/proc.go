package main

import (
	"context"
	"os"

	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/internal/app/setuid"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
)

func mustRunApp(ctx context.Context, config *fst.Config, beforeFail func()) {
	rs := new(fst.RunState)
	a := setuid.MustNew(ctx, std)

	var code int
	if sa, err := a.Seal(config); err != nil {
		fmsg.PrintBaseError(err, "cannot seal app:")
		code = 1
	} else {
		code = setuid.PrintRunStateErr(rs, sa.Run(rs))
	}

	if code != 0 {
		beforeFail()
		os.Exit(code)
	}
}
