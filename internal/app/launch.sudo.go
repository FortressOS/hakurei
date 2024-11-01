package app

import (
	"git.ophivana.moe/security/fortify/internal/fmsg"
)

const (
	sudoAskPass = "SUDO_ASKPASS"
)

func (a *app) commandBuilderSudo(shimEnv string) (args []string) {
	args = make([]string, 0, 8)

	// -Hiu $USER
	args = append(args, "-Hiu", a.seal.sys.user.Username)

	// -A?
	if _, ok := a.os.LookupEnv(sudoAskPass); ok {
		fmsg.VPrintln(sudoAskPass, "set, adding askpass flag")
		args = append(args, "-A")
	}

	// shim payload
	args = append(args, shimEnv)

	// -- $@
	args = append(args, "--", a.os.FshimPath())

	return
}
