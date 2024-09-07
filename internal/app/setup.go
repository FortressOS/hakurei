package app

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"strconv"

	"git.ophivana.moe/cat/fortify/internal/state"
	"git.ophivana.moe/cat/fortify/internal/system"
)

type App struct {
	uid     int
	env     []string
	command []string

	enablements state.Enablements
	*user.User

	// absolutely *no* method of this type is thread-safe
	// so don't treat it as if it is
}

func (a *App) setEnablement(e state.Enablement) {
	if a.enablements.Has(e) {
		panic("enablement " + e.String() + " set twice")
	}

	a.enablements |= e.Mask()
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
