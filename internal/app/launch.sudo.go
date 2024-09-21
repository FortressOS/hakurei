package app

import (
	"os"

	"git.ophivana.moe/cat/fortify/internal/verbose"
)

const (
	sudoAskPass = "SUDO_ASKPASS"
)

func (a *app) commandBuilderSudo() (args []string) {
	args = make([]string, 0, 4+len(a.seal.env)+len(a.seal.command))

	// -Hiu $USER
	args = append(args, "-Hiu", a.seal.sys.Username)

	// -A?
	if _, ok := os.LookupEnv(sudoAskPass); ok {
		verbose.Printf("%s set, adding askpass flag\n", sudoAskPass)
		args = append(args, "-A")
	}

	// environ
	args = append(args, a.seal.env...)

	// -- $@
	args = append(args, "--")
	args = append(args, a.seal.command...)

	return
}
