package main

import (
	"context"
	"os"

	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/internal/app"
	"git.gensokyo.uk/security/fortify/internal/app/instance"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
)

func mustRunApp(ctx context.Context, config *fst.Config, beforeFail func()) {
	rs := new(app.RunState)
	a := instance.MustNew(instance.ISetuid, ctx, std)

	var code int
	if sa, err := a.Seal(config); err != nil {
		fmsg.PrintBaseError(err, "cannot seal app:")
		code = 1
	} else {
		code = instance.PrintRunStateErr(instance.ISetuid, rs, sa.Run(rs))
	}

	if code != 0 {
		beforeFail()
		os.Exit(code)
	}
}
