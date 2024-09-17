package app

import (
	"fmt"
	"os"

	"git.ophivana.moe/cat/fortify/internal"
	"git.ophivana.moe/cat/fortify/internal/verbose"
	"git.ophivana.moe/cat/fortify/xcb"
)

const display = "DISPLAY"

func (a *App) ShareX() {
	a.setEnablement(internal.EnableX)

	// discovery X11 and grant user permission via the `ChangeHosts` command
	if d, ok := os.LookupEnv(display); !ok {
		internal.Fatal("X11: DISPLAY not set")
	} else {
		// add environment variable for new process
		a.AppendEnv(display, d)

		verbose.Printf("X11: Adding XHost entry SI:localuser:%s to display '%s'\n", a.Username, d)
		if err := xcb.ChangeHosts(xcb.HostModeInsert, xcb.FamilyServerInterpreted, "localuser\x00"+a.Username); err != nil {
			internal.Fatal(fmt.Sprintf("Error adding XHost entry to '%s':", d), err)
		} else {
			a.exit.XcbActionComplete()
		}
	}
}
