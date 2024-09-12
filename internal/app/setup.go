package app

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"strconv"

	"git.ophivana.moe/cat/fortify/internal/state"
	"git.ophivana.moe/cat/fortify/internal/util"
	"git.ophivana.moe/cat/fortify/internal/verbose"
)

type App struct {
	launchOptionText string

	uid     int
	env     []string
	command []string

	launchOption uint8
	toolPath     string

	enablements state.Enablements
	*user.User

	// absolutely *no* method of this type is thread-safe
	// so don't treat it as if it is
}

func (a *App) LaunchOption() uint8 {
	return a.launchOption
}

func (a *App) setEnablement(e state.Enablement) {
	if a.enablements.Has(e) {
		panic("enablement " + e.String() + " set twice")
	}

	a.enablements |= e.Mask()
}

func New(userName string, args []string, launchOptionText string) *App {
	a := &App{command: args, launchOptionText: launchOptionText}

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

	verbose.Println("Running as user", a.Username, "("+a.Uid+"),", "command:", a.command)
	if util.SdBootedV {
		verbose.Println("System booted with systemd as init system (PID 1).")
	}

	switch a.launchOptionText {
	case "sudo":
		a.launchOption = LaunchMethodSudo
		if sudoPath, ok := util.Which("sudo"); !ok {
			fmt.Println("Did not find 'sudo' in PATH")
			os.Exit(1)
		} else {
			a.toolPath = sudoPath
		}
	case "bubblewrap":
		a.launchOption = LaunchMethodBwrap
		if bwrapPath, ok := util.Which("bwrap"); !ok {
			fmt.Println("Did not find 'bwrap' in PATH")
			os.Exit(1)
		} else {
			a.toolPath = bwrapPath
		}
	case "systemd":
		a.launchOption = LaunchMethodMachineCtl
		if !util.SdBootedV {
			fmt.Println("System has not been booted with systemd as init system (PID 1).")
			os.Exit(1)
		}

		if machineCtlPath, ok := util.Which("machinectl"); !ok {
			fmt.Println("Did not find 'machinectl' in PATH")
		} else {
			a.toolPath = machineCtlPath
		}
	default:
		fmt.Println("invalid launch method")
		os.Exit(1)
	}

	verbose.Println("Determined launch method to be", a.launchOptionText, "with tool at", a.toolPath)

	return a
}
