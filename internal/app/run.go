package app

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"git.ophivana.moe/cat/fortify/internal/state"
	"git.ophivana.moe/cat/fortify/internal/system"
	"git.ophivana.moe/cat/fortify/internal/util"
)

const (
	term        = "TERM"
	sudoAskPass = "SUDO_ASKPASS"
)
const (
	LaunchMethodSudo = iota
	LaunchMethodMachineCtl

	LaunchBare
	launchOptionLength
)

var (
	// LaunchOptions is set in main's cli.go
	LaunchOptions [launchOptionLength]bool
)

func (a *App) Run() {
	// pass $TERM to launcher
	if t, ok := os.LookupEnv(term); ok {
		a.AppendEnv(term, t)
	}

	commandBuilder := a.commandBuilderSudo

	var toolPath string

	// dependency checks
	const sudoFallback = "Falling back to 'sudo', some desktop integration features may not work"
	if LaunchOptions[LaunchMethodMachineCtl] && !LaunchOptions[LaunchMethodSudo] { // sudo argument takes priority
		if !util.SdBooted() {
			fmt.Println("This system was not booted through systemd")
			fmt.Println(sudoFallback)
		} else if machineCtlPath, ok := util.Which("machinectl"); !ok {
			fmt.Println("Did not find 'machinectl' in PATH")
			fmt.Println(sudoFallback)
		} else {
			toolPath = machineCtlPath
			commandBuilder = a.commandBuilderMachineCtl
		}
	} else if sudoPath, ok := util.Which("sudo"); !ok {
		state.Fatal("Did not find 'sudo' in PATH")
	} else {
		toolPath = sudoPath
	}

	if system.V.Verbose {
		fmt.Printf("Selected launcher '%s' bare=%t\n", toolPath, LaunchOptions[LaunchBare])
	}

	cmd := exec.Command(toolPath, commandBuilder(LaunchOptions[LaunchBare])...)
	cmd.Env = a.env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = system.V.RunDir

	if system.V.Verbose {
		fmt.Println("Executing:", cmd)
	}

	if err := cmd.Start(); err != nil {
		state.Fatal("Error starting process:", err)
	}

	state.RegisterEnablement(a.enablements)

	if err := state.SaveProcess(a.Uid, cmd); err != nil {
		// process already started, shouldn't be fatal
		fmt.Println("Error registering process:", err)
	}

	var r int
	if err := cmd.Wait(); err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) {
			state.Fatal("Error running process:", err)
		}
	}

	if system.V.Verbose {
		fmt.Println("Process exited with exit code", r)
	}
	state.BeforeExit()
	os.Exit(r)
}

func (a *App) commandBuilderSudo(bare bool) (args []string) {
	args = make([]string, 0, 4+len(a.env)+len(a.command))

	// -Hiu $USER
	args = append(args, "-Hiu", a.Username)

	// -A?
	if _, ok := os.LookupEnv(sudoAskPass); ok {
		if system.V.Verbose {
			fmt.Printf("%s set, adding askpass flag\n", sudoAskPass)
		}
		args = append(args, "-A")
	}

	// environ
	args = append(args, a.env...)

	// -- $@
	args = append(args, "--")
	args = append(args, a.command...)

	return
}

func (a *App) commandBuilderMachineCtl(bare bool) (args []string) {
	args = make([]string, 0, 9+len(a.env))

	// shell --uid=$USER
	args = append(args, "shell", "--uid="+a.Username)

	// --quiet
	if !system.V.Verbose {
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
		state.Fatal("Did not find 'sh' in PATH")
	} else {
		args = append(args, sh, "-c")
	}

	if len(a.command) == 0 { // execute shell if command is not provided
		a.command = []string{"$SHELL"}
	}

	innerCommand := strings.Builder{}

	if !bare {
		innerCommand.WriteString("dbus-update-activation-environment --systemd")
		for _, e := range a.env {
			innerCommand.WriteString(" " + strings.SplitN(e, "=", 2)[0])
		}
		innerCommand.WriteString("; ")
		//innerCommand.WriteString("systemctl --user start xdg-desktop-portal-gtk; ")
	}

	if executable, err := os.Executable(); err != nil {
		state.Fatal("Error reading executable path:", err)
	} else {
		innerCommand.WriteString("exec " + executable + " -V")
	}
	args = append(args, innerCommand.String())

	return
}
