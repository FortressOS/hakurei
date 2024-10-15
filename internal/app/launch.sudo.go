package app

import (
	"os"

	"git.ophivana.moe/cat/fortify/internal/verbose"
)

const (
	sudoAskPass = "SUDO_ASKPASS"
)

func (a *app) commandBuilderSudo(shimEnv string) (args []string) {
	args = make([]string, 0, 8)

	// -Hiu $USER
	args = append(args, "-Hiu", a.seal.sys.user.Username)

	// -A?
	if _, ok := os.LookupEnv(sudoAskPass); ok {
		verbose.Printf("%s set, adding askpass flag\n", sudoAskPass)
		args = append(args, "-A")
	}

	// shim payload
	args = append(args, shimEnv)

	// -- $@
	args = append(args, "--", a.seal.sys.executable, "shim")

	return
}
