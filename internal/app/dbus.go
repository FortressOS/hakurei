package app

import (
	"fmt"

	"git.ophivana.moe/cat/fortify/internal/state"
)

func (a *App) ShareDBus() {
	a.setEnablement(state.EnableDBus)

	// TODO: start xdg-dbus-proxy
	fmt.Println("warn: dbus proxy not implemented")
}
