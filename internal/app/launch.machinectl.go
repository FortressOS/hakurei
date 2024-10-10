package app

import (
	"os/exec"
	"strings"

	"git.ophivana.moe/cat/fortify/internal/state"
	"git.ophivana.moe/cat/fortify/internal/verbose"
)

func (a *app) commandBuilderMachineCtl(shimEnv string) (args []string) {
	args = make([]string, 0, 9+len(a.seal.env))

	// shell --uid=$USER
	args = append(args, "shell", "--uid="+a.seal.sys.Username)

	// --quiet
	if !verbose.Get() {
		args = append(args, "--quiet")
	}

	// environ
	envQ := make([]string, len(a.seal.env)+1)
	for i, e := range a.seal.env {
		envQ[i] = "-E" + e
	}
	// add shim payload to environment for shim path
	envQ[len(a.seal.env)] = "-E" + shimEnv
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
	for _, e := range a.seal.env {
		innerCommand.WriteString(" " + strings.SplitN(e, "=", 2)[0])
	}
	innerCommand.WriteString("; ")

	// override message bus address if enabled
	if a.seal.et.Has(state.EnableDBus) {
		innerCommand.WriteString(dbusSessionBusAddress + "=" + "'" + "unix:path=" + a.seal.sys.dbusAddr[0][1] + "' ")
		if a.seal.sys.dbusSystem {
			innerCommand.WriteString(dbusSystemBusAddress + "=" + "'" + "unix:path=" + a.seal.sys.dbusAddr[1][1] + "' ")
		}
	}

	// both license and version flags need to be set to activate shim path
	innerCommand.WriteString("exec " + a.seal.sys.executable + " -V -license")

	// append inner command
	args = append(args, innerCommand.String())

	return
}
