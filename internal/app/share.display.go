package app

import (
	"errors"
	"os"
	"path"

	"git.ophivana.moe/cat/fortify/acl"
	"git.ophivana.moe/cat/fortify/internal/state"
)

const (
	term    = "TERM"
	display = "DISPLAY"

	// https://manpages.debian.org/experimental/libwayland-doc/wl_display_connect.3.en.html
	waylandDisplay = "WAYLAND_DISPLAY"
)

var (
	ErrWayland  = errors.New(waylandDisplay + " unset")
	ErrXDisplay = errors.New(display + " unset")
)

type ErrDisplayEnv BaseError

func (seal *appSeal) shareDisplay() error {
	// pass $TERM to launcher
	if t, ok := os.LookupEnv(term); ok {
		seal.appendEnv(term, t)
	}

	// set up wayland
	if seal.et.Has(state.EnableWayland) {
		if wd, ok := os.LookupEnv(waylandDisplay); !ok {
			return (*ErrDisplayEnv)(wrapError(ErrWayland, "WAYLAND_DISPLAY is not set"))
		} else if seal.wlDone == nil {
			// hardlink wayland socket
			wp := path.Join(seal.RuntimePath, wd)
			wpi := path.Join(seal.shareLocal, "wayland")
			seal.sys.link(wp, wpi)
			seal.appendEnv(waylandDisplay, wpi)

			// ensure Wayland socket ACL (e.g. `/run/user/%d/wayland-%d`)
			seal.sys.updatePermTag(state.EnableWayland, wp, acl.Read, acl.Write, acl.Execute)
		} else {
			// set wayland socket path (e.g. `/run/user/%d/wayland-%d`)
			seal.wl = path.Join(seal.RuntimePath, wd)
		}
	}

	// set up X11
	if seal.et.Has(state.EnableX) {
		// discover X11 and grant user permission via the `ChangeHosts` command
		if d, ok := os.LookupEnv(display); !ok {
			return (*ErrDisplayEnv)(wrapError(ErrXDisplay, "DISPLAY is not set"))
		} else {
			seal.sys.changeHosts(seal.sys.Username)
			seal.appendEnv(display, d)
		}
	}

	return nil
}
