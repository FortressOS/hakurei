package app

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"git.ophivana.moe/cat/fortify/internal"
	"git.ophivana.moe/cat/fortify/internal/state"
	"git.ophivana.moe/cat/fortify/internal/util"
	"git.ophivana.moe/cat/fortify/internal/verbose"
)

const (
	term        = "TERM"
	sudoAskPass = "SUDO_ASKPASS"
)
const (
	LaunchMethodSudo uint8 = iota
	LaunchMethodBwrap
	LaunchMethodMachineCtl
)

func (a *App) Run() {
	// pass $TERM to launcher
	if t, ok := os.LookupEnv(term); ok {
		a.AppendEnv(term, t)
	}

	var commandBuilder func() (args []string)

	switch a.launchOption {
	case LaunchMethodSudo:
		commandBuilder = a.commandBuilderSudo
	case LaunchMethodBwrap:
		commandBuilder = a.commandBuilderBwrap
	case LaunchMethodMachineCtl:
		commandBuilder = a.commandBuilderMachineCtl
	default:
		panic("unreachable")
	}

	cmd := exec.Command(a.toolPath, commandBuilder()...)
	cmd.Env = []string{}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = a.runDirPath

	verbose.Println("Executing:", cmd)

	if err := cmd.Start(); err != nil {
		internal.Fatal("Error starting process:", err)
	}

	a.exit.SealEnablements(a.enablements)

	if statePath, err := state.SaveProcess(a.Uid, cmd, a.runDirPath, a.command, a.enablements); err != nil {
		// process already started, shouldn't be fatal
		fmt.Println("Error registering process:", err)
	} else {
		a.exit.SealStatePath(statePath)
	}

	var r int
	if err := cmd.Wait(); err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) {
			internal.Fatal("Error running process:", err)
		}
	}

	verbose.Println("Process exited with exit code", r)
	internal.BeforeExit()
	os.Exit(r)
}

func (a *App) commandBuilderSudo() (args []string) {
	args = make([]string, 0, 4+len(a.env)+len(a.command))

	// -Hiu $USER
	args = append(args, "-Hiu", a.Username)

	// -A?
	if _, ok := os.LookupEnv(sudoAskPass); ok {
		verbose.Printf("%s set, adding askpass flag\n", sudoAskPass)
		args = append(args, "-A")
	}

	// environ
	args = append(args, a.env...)

	// -- $@
	args = append(args, "--")
	args = append(args, a.command...)

	return
}

func (a *App) commandBuilderBwrap() (args []string) {
	// TODO: build bwrap command
	internal.Fatal("bwrap")
	panic("unreachable")
}

func (a *App) commandBuilderMachineCtl() (args []string) {
	args = make([]string, 0, 9+len(a.env))

	// shell --uid=$USER
	args = append(args, "shell", "--uid="+a.Username)

	// --quiet
	if !verbose.Get() {
		args = append(args, "--quiet")
	}

	// environ
	envQ := make([]string, len(a.env)+1)
	for i, e := range a.env {
		envQ[i] = "-E" + e
	}
	envQ[len(a.env)] = "-E" + a.launcherPayloadEnv()
	args = append(args, envQ...)

	// -- .host
	args = append(args, "--", ".host")

	// /bin/sh -c
	if sh, ok := util.Which("sh"); !ok {
		internal.Fatal("Did not find 'sh' in PATH")
	} else {
		args = append(args, sh, "-c")
	}

	if len(a.command) == 0 { // execute shell if command is not provided
		a.command = []string{"$SHELL"}
	}

	innerCommand := strings.Builder{}

	innerCommand.WriteString("dbus-update-activation-environment --systemd")
	for _, e := range a.env {
		innerCommand.WriteString(" " + strings.SplitN(e, "=", 2)[0])
	}
	innerCommand.WriteString("; ")

	if executable, err := os.Executable(); err != nil {
		internal.Fatal("Error reading executable path:", err)
	} else {
		if a.enablements.Has(internal.EnableDBus) {
			innerCommand.WriteString(dbusSessionBusAddress + "=" + "'" + dbusAddress[0] + "' ")
			if dbusSystem {
				innerCommand.WriteString(dbusSystemBusAddress + "=" + "'" + dbusAddress[1] + "' ")
			}
		}
		innerCommand.WriteString("exec " + executable + " -V")
	}
	args = append(args, innerCommand.String())

	return
}
