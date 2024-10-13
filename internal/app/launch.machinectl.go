package app

import (
	"os/exec"
	"strings"

	"git.ophivana.moe/cat/fortify/internal/state"
	"git.ophivana.moe/cat/fortify/internal/verbose"
)

func (a *app) commandBuilderMachineCtl(shimEnv string) (args []string) {
	args = make([]string, 0, 9+len(a.seal.sys.bwrap.SetEnv))

	// shell --uid=$USER
	args = append(args, "shell", "--uid="+a.seal.sys.Username)

	// --quiet
	if !verbose.Get() {
		args = append(args, "--quiet")
	}

	// environ
	envQ := make([]string, 0, len(a.seal.sys.bwrap.SetEnv)+1)
	for k, v := range a.seal.sys.bwrap.SetEnv {
		envQ = append(envQ, "-E"+k+"="+v)
	}
	// add shim payload to environment for shim path
	envQ = append(envQ, "-E"+shimEnv)
	args = append(args, envQ...)

	// -- .host
	args = append(args, "--", ".host")

	// /bin/sh -c
	if sh, err := exec.LookPath("sh"); err != nil {
		// hardcode /bin/sh path since it exists more often than not
		args = append(args, "/bin/sh", "-c")
	} else {
		args = append(args, sh, "-c")
	}

	// build inner command expression ran as target user
	innerCommand := strings.Builder{}

	// apply custom environment variables to activation environment
	innerCommand.WriteString("dbus-update-activation-environment --systemd")
	for k := range a.seal.sys.bwrap.SetEnv {
		innerCommand.WriteString(" " + k)
	}
	innerCommand.WriteString("; ")

	// override message bus address if enabled
	if a.seal.et.Has(state.EnableDBus) {
		innerCommand.WriteString(dbusSessionBusAddress + "=" + "'" + "unix:path=" + a.seal.sys.dbusAddr[0][1] + "' ")
		if a.seal.sys.dbusSystem {
			innerCommand.WriteString(dbusSystemBusAddress + "=" + "'" + "unix:path=" + a.seal.sys.dbusAddr[1][1] + "' ")
		}
	}

	// launch fortify as shim
	innerCommand.WriteString("exec " + a.seal.sys.executable + " shim")

	// append inner command
	args = append(args, innerCommand.String())

	return
}
