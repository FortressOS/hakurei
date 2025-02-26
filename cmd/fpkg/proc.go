package main

import (
	"context"
	"os"

	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/internal/app"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
)

func mustRunApp(ctx context.Context, config *fst.Config, beforeFail func()) {
	rs := new(fst.RunState)
	a := app.MustNew(std)

	if sa, err := a.Seal(config); err != nil {
		fmsg.PrintBaseError(err, "cannot seal app:")
		rs.ExitCode = 1
	} else {
		// this updates ExitCode
		app.PrintRunStateErr(rs, sa.Run(ctx, rs))
	}

	if rs.ExitCode != 0 {
		beforeFail()
		os.Exit(rs.ExitCode)
	}
}
