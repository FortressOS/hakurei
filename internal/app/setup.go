package app

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"

	"git.ophivana.moe/cat/fortify/internal/state"
	"git.ophivana.moe/cat/fortify/internal/system"
	"git.ophivana.moe/cat/fortify/internal/util"
)

type App struct {
	uid     int
	env     []string
	command []string

	*user.User
}

func (a *App) Run() {
	f := a.launchBySudo
	m, b := false, false
	switch {
	case system.MethodFlags[0]: // sudo
	case system.MethodFlags[1]: // bare
		m, b = true, true
	default: // machinectl
		m, b = true, false
	}

	var toolPath string

	// dependency checks
	const sudoFallback = "Falling back to 'sudo', some desktop integration features may not work"
	if m {
		if !util.SdBooted() {
			fmt.Println("This system was not booted through systemd")
			fmt.Println(sudoFallback)
		} else if tp, ok := util.Which("machinectl"); !ok {
			fmt.Println("Did not find 'machinectl' in PATH")
			fmt.Println(sudoFallback)
		} else {
			toolPath = tp
			f = func() []string { return a.launchByMachineCtl(b) }
		}
	} else if tp, ok := util.Which("sudo"); !ok {
		state.Fatal("Did not find 'sudo' in PATH")
	} else {
		toolPath = tp
	}

	if system.V.Verbose {
		fmt.Printf("Selected launcher '%s' bare=%t\n", toolPath, b)
	}

	cmd := exec.Command(toolPath, f()...)
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

func New(userName string, args []string) *App {
	a := &App{command: args}

	if u, err := user.Lookup(userName); err != nil {
		if errors.As(err, new(user.UnknownUserError)) {
			fmt.Println("unknown user", userName)
		} else {
			// unreachable
			panic(err)
		}

		// too early for fatal
		os.Exit(1)
	} else {
		a.User = u
	}

	if u, err := strconv.Atoi(a.Uid); err != nil {
		// usually unreachable
		panic("uid parse")
	} else {
		a.uid = u
	}

	if system.V.Verbose {
		fmt.Println("Running as user", a.Username, "("+a.Uid+"),", "command:", a.command)
	}

	return a
}
